package kafkaclient

import (
	"context"
	"fmt"
	"reflect"

	"github.com/Shopify/sarama"
	logger "github.com/disturb16/apilogger"
)

type saramaProducer struct {
	producer    interface{}
	topicConf   map[string]TopicConfig
	avroCodec   EncoderDecoder
	jsonCodec   EncoderDecoder
	stringCodec EncoderDecoder
}

func newSaramaProducer(ctx context.Context,
	prodType producerType, brokers []string, topicConf map[string]TopicConfig,
	saramaConf *sarama.Config) (p saramaProducer, e error) {

	lg := logger.New(ctx, "")

	switch prodType {
	case ProducerTypeSync:
		p.producer, e = sarama.NewSyncProducer(brokers, saramaConf)
	case ProducerTypeAsync:
		p.producer, e = sarama.NewAsyncProducer(brokers, saramaConf)
	default:
		e = errInvalidProducer
	}

	if e != nil {
		lg.Error(logger.LogCatUncategorized, e)
	}

	p.topicConf = topicConf
	return
}

func (p *saramaProducer) produceMessage(
	ctx context.Context, topic string, key string, msg interface{}) (e error) {

	lg := logger.New(ctx, "")

	se, e := p.getSaramaEncoder(ctx, p.topicConf[topic], msg)
	if e != nil {
		lg.Error(logger.LogCatUncategorized, e)
		return
	}

	m := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(key),
		Value: se}

	switch p.producer.(type) {
	case sarama.SyncProducer:
		_, _, e = (p.producer.(sarama.SyncProducer)).SendMessage(m)
	case sarama.AsyncProducer:
		(p.producer.(sarama.AsyncProducer)).Input() <- m
	default:
		return errInvalidProducer
	}

	if e != nil {
		lg.Error(logger.LogCatUncategorized, e)
	}

	return
}

func (p *saramaProducer) getSaramaEncoder(ctx context.Context,
	topicConf TopicConfig, msg interface{}) (s sarama.Encoder, e error) {

	lg := logger.New(ctx, "")

	var codec EncoderDecoder

	switch topicConf.MessageType {
	case MessageFormatAvro:
		codec = p.avroCodec
		break
	case MessageFormatJSON:
		codec = p.jsonCodec
		break
	case MessageFormatString:
		codec = p.stringCodec
		break
	default:
		e = errMessageFormat
		lg.Error(logger.LogCatUncategorized, e)
		return
	}

	switch msg.(type) {
	case string:
		s = sarama.StringEncoder(msg.(string))
		break
	case []byte:
		s = newSaramaByteEncoder(ctx, topicConf.Name, msg.([]byte), codec)
		break
	case int32, int64, float32, float64:
		s = sarama.StringEncoder(fmt.Sprint(msg))
	default:
		if reflect.ValueOf(msg).Kind() == reflect.Struct {
			s, e = newSaramaStructEncoder(ctx, topicConf.Name, msg, codec)
			break
		}
		e = errMessageType
	}

	if e != nil {
		lg.Error(logger.LogCatUncategorized, e)
	}

	return
}

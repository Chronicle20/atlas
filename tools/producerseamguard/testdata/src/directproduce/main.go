package directproduce

import "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

func bad() {
	_ = producer.Produce(nil) // want `direct producer.Produce outside libs/`
}

package filter

import (
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type Bloom struct {
	filter    filter.Filter
	generator filter.FilterGenerator
	buf       []byte
}

func New() *Bloom {
	b := &Bloom{
		filter: filter.NewBloomFilter(10),
	}
	b.generator = b.filter.NewGenerator()
	return b
}

func (b *Bloom) CheckOrAdd(key []byte) (exist bool) {
	b.build()
	if b.filter.Contains(b.buf, key) {
		return true
	}
	b.generator.Add(key)
	return false
}

func (b *Bloom) build() {
	buf := &util.Buffer{}
	b.generator.Generate(buf)
	b.buf = buf.Bytes()
}

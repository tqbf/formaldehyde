package ewma

import (
	"math"
)

type EWMA struct {
	internal uint64
	weight   uint
	factor   uint
}

// cadged shamelessly from Linux 

func (self *EWMA) Add(sample uint64) uint64 {
	if self.internal == 0 {
		if self.weight == 0 {
			self.weight = 1024
		}
		self.weight = uint(math.Ilogb(float64(self.weight)))

		if self.factor == 0 { 
			self.factor = 8
		} 
		self.factor = uint(math.Ilogb(float64(self.factor)))

		self.internal = (sample << self.factor)
	} else {
		self.internal = (((self.internal << self.weight) - self.internal) +
			(sample << self.factor)) >> self.weight

	}

	return self.internal >> self.factor
}

func (self *EWMA) Read() uint64 {
	return self.internal >> self.factor
}

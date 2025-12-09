package uid

import (
	"errors"
	"sync"
	"time"
)

// Snowflake constants
const (
	epoch             = int64(1672531200000) // 2023-01-01 00:00:00 UTC
	workerIDBits      = 5
	datacenterIDBits  = 5
	sequenceBits      = 12
	maxWorkerID       = -1 ^ (-1 << workerIDBits)
	maxDatacenterID   = -1 ^ (-1 << datacenterIDBits)
	maxSequence       = -1 ^ (-1 << sequenceBits)
	workerIDShift     = sequenceBits
	datacenterIDShift = sequenceBits + workerIDBits
	timestampShift    = sequenceBits + workerIDBits + datacenterIDBits
)

type Snowflake struct {
	mu           sync.Mutex
	timestamp    int64
	workerID     int64
	datacenterID int64
	sequence     int64
}

func NewSnowflake(workerID, datacenterID int64) (*Snowflake, error) {
	if workerID < 0 || workerID > maxWorkerID {
		return nil, errors.New("worker ID out of range")
	}
	if datacenterID < 0 || datacenterID > maxDatacenterID {
		return nil, errors.New("datacenter ID out of range")
	}

	return &Snowflake{
		timestamp:    0,
		workerID:     workerID,
		datacenterID: datacenterID,
		sequence:     0,
	}, nil
}

func (s *Snowflake) NextID() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()

	if now < s.timestamp {
		return 0, errors.New("clock moved backwards")
	}

	if now == s.timestamp {
		s.sequence = (s.sequence + 1) & maxSequence
		if s.sequence == 0 {
			for now <= s.timestamp {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		s.sequence = 0
	}

	s.timestamp = now

	id := ((now - epoch) << timestampShift) |
		(s.datacenterID << datacenterIDShift) |
		(s.workerID << workerIDShift) |
		s.sequence

	return id, nil
}



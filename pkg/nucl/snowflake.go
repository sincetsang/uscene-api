package nucl

import (
	"errors"
	"sync"
	"time"
)

type Snowflake struct {
	mutex       sync.Mutex
	lastTime    int64
	sequence    int64
	machineID   int64
	timeBits    int
	machineBits int
	seqBits     int
	maxSequence int64
	maxMachine  int64
	epoch       int64
}

// initSnowflake 创建 Snowflake 实例
func (nu *Nucleus) initSnowflake() error {
	machineID := int64(1)
	epoch := int64(1704067200)
	timeBits := 32
	machineBits := 5
	seqBits := 16

	maxMachine := int64((1 << machineBits) - 1)
	if machineID > maxMachine || machineID < 0 {
		return errors.New("machine ID out of range")
	}

	maxSequence := int64((1 << seqBits) - 1)
	nu.Snowflake = &Snowflake{
		lastTime:    0,
		sequence:    0,
		machineID:   machineID,
		timeBits:    timeBits,
		machineBits: machineBits,
		seqBits:     seqBits,
		maxSequence: maxSequence,
		maxMachine:  maxMachine,
		epoch:       epoch,
	}
	return nil
}

func (sf *Snowflake) GenerateID() int64 {
	sf.mutex.Lock()
	defer sf.mutex.Unlock()

	now := time.Now().Unix() // 秒级时间戳
	if now < sf.lastTime {
		// 使用逻辑时钟，模拟时间前进
		now = sf.lastTime + 1
	}

	if now == sf.lastTime {
		// 如果在同一秒内，增加序列号
		sf.sequence = (sf.sequence + 1) & sf.maxSequence
		if sf.sequence == 0 {
			// 如果序列号溢出，等待下一秒
			for now <= sf.lastTime {
				now = time.Now().Unix()
			}
		}
	} else {
		// 时间戳变化，重置序列号
		sf.sequence = 0
	}

	sf.lastTime = now

	// 生成 ID：| 时间戳 | 机器 ID | 序列号 |
	timeShift := sf.machineBits + sf.seqBits
	machineShift := sf.seqBits

	return ((now - sf.epoch) << timeShift) | (sf.machineID << machineShift) | sf.sequence
}

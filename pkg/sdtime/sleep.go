package sdtime

import (
	"time"
)

func SleepM(minutes int64) {
	time.Sleep(Minutes(minutes))
}

func SleepS(seconds int64) {
	time.Sleep(Seconds(seconds))
}

func SleepMs(ms int64) {
	time.Sleep(Millis(ms))
}

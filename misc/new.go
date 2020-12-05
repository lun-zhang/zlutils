package misc

import "time"

func NewInt(i int) *int       { return &i }
func NewInt8(i int8) *int8    { return &i }
func NewInt16(i int16) *int16 { return &i }
func NewInt32(i int32) *int32 { return &i }
func NewInt64(i int64) *int64 { return &i }

func NewUint(i uint) *uint       { return &i }
func NewUint8(i uint8) *uint8    { return &i }
func NewUint16(i uint16) *uint16 { return &i }
func NewUint32(i uint32) *uint32 { return &i }
func NewUint64(i uint64) *uint64 { return &i }

func NewFloat32(i float32) *float32 { return &i }
func NewFloat64(i float64) *float64 { return &i }

func NewString(i string) *string { return &i }
func NewByte(i byte) *byte       { return &i }
func NewRune(i rune) *rune       { return &i }

func NewTime(i time.Time) *time.Time             { return &i }
func NewDuration(i time.Duration) *time.Duration { return &i }

func NewBool(i bool) *bool { return &i }

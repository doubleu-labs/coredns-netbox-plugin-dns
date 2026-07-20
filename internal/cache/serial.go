package cache

const serialHalfRange = uint32(1 << 31)

func serialBefore(a, b uint32) bool {
	return (a < b && b-a < serialHalfRange) || (a > b && a-b > serialHalfRange)
}

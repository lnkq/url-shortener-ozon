package shortcode

const Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_"

const Length = 10

const base = uint64(len(Alphabet))

func Encode(id uint64) string {
	buf := make([]byte, Length)
	for i := Length - 1; i >= 0; i-- {
		buf[i] = Alphabet[id%base]
		id /= base
	}
	return string(buf)
}

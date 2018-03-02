package crypto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"sync"

	"github.com/rogpeppe/fastuuid"

	"go_commons/utils"
)

const (
	IvSize        = 8
	TextSize      = 8
	SignatureSize = 4
	TSize         = IvSize + TextSize + SignatureSize

	IvOffset        = 0
	TextOffset      = IvOffset + IvSize
	SignatureOffset = TextOffset + TextSize
)

type Decrypter struct {
	lk sync.Mutex
	*fastuuid.Generator

	Ekey []byte // 加密 key
	Ikey []byte // 校验 key
}

func New(ekey, ikey string) (decrypter *Decrypter, err error) {
	ek, err := hex.DecodeString(ekey)
	if err != nil {
		err = errors.New("invaild ekey:" + err.Error())
		return
	}
	ik, err := hex.DecodeString(ikey)
	if err != nil {
		err = errors.New("invaild ikey:" + err.Error())
		return
	}

	g, err := fastuuid.NewGenerator()
	if err != nil {
		return
	}

	decrypter = &Decrypter{
		Generator: g,

		Ekey: ek,
		Ikey: ik,
	}
	return
}

func (this *Decrypter) Decode(src string) (text int64, err error) {
	//base64 decode
	dest, err := utils.Base64UrlDecoder(src)
	if err != nil {
		return
	}

	if len(dest) != TSize {
		err = errors.New("base64 decode error: " + src)
		return
	}

	iv := dest[IvOffset:TextOffset]
	encText := dest[TextOffset:SignatureOffset]
	signature := dest[SignatureOffset:]
	pad := utils.HmacEncoder(this.Ekey, iv)

	bText := make([]byte, TextSize, TextSize+IvSize)
	for i := 0; i < TextSize; i++ {
		bText[i] = encText[i] ^ pad[i]
	}
	r := bytes.NewBuffer(bText)
	if err = binary.Read(r, binary.BigEndian, &text); err != nil {
		return
	}

	bText = append(bText, iv...)
	confSig := utils.HmacEncoder(this.Ikey, bText)[:SignatureSize]

	if !bytes.Equal(confSig, signature) {
		err = errors.New("bad signature, src:" + src)
	}
	return
}

func (this *Decrypter) Encode(text int64) (eText string, err error) {
	this.lk.Lock()
	uid := this.Next()
	this.lk.Unlock()

	buf, iv := new(bytes.Buffer), uid[:IvSize]
	if err = binary.Write(buf, binary.BigEndian, text); err != nil {
		return
	}
	bText := buf.Bytes()

	encText := make([]byte, TextSize)
	pad := utils.HmacEncoder(this.Ekey, iv)
	for i := 0; i < TextSize; i++ {
		encText[i] = pad[i] ^ bText[i]
	}

	bText = append(bText, iv...)
	signature := utils.HmacEncoder(this.Ikey, bText)[:SignatureSize] //take first 4 bytes

	iv = append(iv, encText...)
	iv = append(iv, signature...)
	eText = utils.Base64UrlEncode(string(iv))
	return
}

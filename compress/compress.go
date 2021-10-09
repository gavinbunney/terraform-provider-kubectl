package compress

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
)

func PackString(str string) string {
	handleErr := func(err error) string {
		e := fmt.Errorf("[WARN] failed to pack string, %v", err)
		log.Printf("%v", e)
		return ""
	}

	buf := &bytes.Buffer{}
	gw, err := flate.NewWriter(buf, flate.BestSpeed)
	defer func() { _ = gw.Close() }()

	if err != nil {
		return handleErr(err)
	}

	data := []byte(str)
	if _, err = gw.Write(data); err != nil {
		return handleErr(err)
	}

	_ = gw.Close()

	// encode in base64 so we can store it in the state file
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func UnpackString(str string) string {
	handleErr := func(err error) string {
		e := fmt.Errorf("[WARN] failed to unpack string, %v", err)
		log.Printf("%v", e)
		return ""
	}

	decoded, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return handleErr(err)
	}

	gr := flate.NewReader(bytes.NewBuffer(decoded))
	defer func() { _ = gr.Close() }()

	data, err := ioutil.ReadAll(gr)
	if err != nil {
		return handleErr(err)
	}

	buf := bytes.Buffer{}
	buf.Write(data)
	return buf.String()
}

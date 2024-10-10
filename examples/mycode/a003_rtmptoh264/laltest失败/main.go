package main

import (
	"fmt"
	"os"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/remux"
	"github.com/q191201771/lal/pkg/rtmp"
)

func main() {
	url := "rtmp://127.0.0.1:1935/live/test111"
	filename := "out.h264"

	var err error
	var w *os.File
	w, err = os.Create(filename)
	if err != nil {
		fmt.Println("创建文件失败", err)
		return
	}
	defer w.Close()

	i := 0
	session := rtmp.NewPullSession(func(option *rtmp.PullSessionOption) {
		option.PullTimeoutMs = 10000
		option.ReadAvTimeoutMs = 10000
		option.ReadBufSize = 0
	}).WithOnReadRtmpAvMsg(func(msg base.RtmpMsg) {
		if filename != "" {
			if i >= NumberOfFrames {
				w.Close()
				os.Exit(0)
				return
			}
			tag := remux.RtmpMsg2FlvTag(msg)
			if tag.Header.Type == 8 {
				return
			} else if tag.Header.Type == 9 {
				w.Write([]byte{0, 0, 0, 1})
				w.Write(msg.Payload)
				i++
			} else {
				w.Write([]byte{0, 0, 0, 1})
				w.Write(msg.Payload)
			}
			// d, _ := json.MarshalIndent(tag, "", "  ")
			// _ = d
			// // fmt.Println(string(d))
			// err := w.WriteTag(*tag)
			// if err != nil {
			// 	fmt.Println("flv文件WriteTag失败", err)
			// }
			// // nazalog.Assert(nil, err)
		}
	})
	err = session.Pull(url)
	if err != nil {
		fmt.Printf("pull failed. err=%+v", err)
		return
	}

	err = <-session.WaitChan()
	if err != nil {
		fmt.Printf("pull WaitChan. err=%+v", err)
	}

}

// const NumberOfFrames = 30 * 20 //30帧 20秒
const NumberOfFrames = 25 * 20 //30帧 20秒

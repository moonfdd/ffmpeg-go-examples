package main

import (
	"fmt"
	"os"
	"time"

	"github.com/deepch/vdk/format/flv"
	"github.com/deepch/vdk/format/rtspv2"
)

func main() {
	// rtmp://mobliestream.c3tv.com:554/live/goodtv.sdp
	// rtmp://liteavapp.qcloud.com/live/liteavdemoplayerstreamid
	// rtmp://127.0.0.1:1935/live/test
	url := "rtsp://127.0.0.1:5544/live/test111"
	filename := "out.flv"
	c, err := rtspv2.Dial(rtspv2.RTSPClientOptions{URL: url, DisableAudio: false, DialTimeout: 10 * time.Second, ReadWriteTimeout: 10 * time.Second, Debug: false})
	if err != nil {
		fmt.Println("rtsp连接失败", err)
		return
	}
	defer c.Close()
	var f1 *os.File
	f1, err = os.Create(filename)
	if err != nil {
		fmt.Println("创建文件失败", err)
		return
	}
	defer f1.Close()
	m := flv.NewMuxer(f1)
	defer m.WriteTrailer()
	i := 0
	for {
		isbreak := false
		select {
		case signals := <-c.Signals:
			switch signals {
			case rtspv2.SignalStreamRTPStop:
				isbreak = true
			}
		case packetAV := <-c.OutgoingPacketQueue:
			codeData := c.CodecData
			// fmt.Println("len(codeData) = ", len(codeData))
			if len(codeData) == 0 {
				fmt.Println("len(codeData) == ", len(codeData))
				return
			}

			if codeData[packetAV.Idx].Type().IsVideo() {

				if i == 0 {
					err = m.WriteHeader(codeData)
					if err != nil {
						fmt.Println("WriteHeader失败", err)
						return
					}
				}
				i++
				err = m.WritePacket(*packetAV)
				if err != nil {
					fmt.Println("WritePacket失败", err)
					isbreak = true
				}
			} else if codeData[packetAV.Idx].Type().IsAudio() && i > 0 {

				err = m.WritePacket(*packetAV)
				if err != nil {
					fmt.Println("WritePacket失败", err)
					isbreak = true
				}
			} else {
				fmt.Println("其他")
			}
		}

		if i >= NumberOfFrames || isbreak {
			break
		}
	}

}

const NumberOfFrames = 25 * 20 //30帧 20秒

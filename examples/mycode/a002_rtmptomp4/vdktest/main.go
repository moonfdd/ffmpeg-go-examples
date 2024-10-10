package main

import (
	"fmt"
	"os"

	"github.com/deepch/vdk/format/mp4"
	"github.com/deepch/vdk/format/rtmp"
)

func main() {
	// rtmp://mobliestream.c3tv.com:554/live/goodtv.sdp
	// rtmp://liteavapp.qcloud.com/live/liteavdemoplayerstreamid
	// rtmp://127.0.0.1:1935/live/test
	url := "rtmp://127.0.0.1:1935/live/test111"
	filename := "out.mp4"
	c, err := rtmp.Dial(url)
	if err != nil {
		fmt.Println("rtmp连接失败", err)
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
	m := mp4.NewMuxer(f1)
	defer m.WriteTrailer()
	i := 0
	for {
		isbreak := false
		packetAV, err2 := c.ReadPacket()
		err = err2
		if err != nil {
			fmt.Println("ReadPacket失败", err)
			isbreak = true
		} else {
			if packetAV.IsKeyFrame {
				fmt.Println("数据：", packetAV.IsKeyFrame, i)
			}
			// fmt.Println("数据：", packetAV.IsKeyFrame, i)
			codeData, err2 := c.Streams()
			err = err2
			if err != nil {
				fmt.Println("Streams失败", err)
				return
			}
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
				err = m.WritePacket(packetAV)
				if err != nil {
					fmt.Println("WritePacket失败", err)
					isbreak = true
				}
			} else if codeData[packetAV.Idx].Type().IsAudio() && i > 0 {

				err = m.WritePacket(packetAV)
				if err != nil {
					fmt.Println("WritePacket失败", err)
					isbreak = true
				}
			}
		}
		if i >= NumberOfFrames || isbreak {
			break
		}
	}

}

const NumberOfFrames = 25 * 20 //30帧 20秒

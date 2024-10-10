package main

import (
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/yapingcat/gomedia/go-codec"
	"github.com/yapingcat/gomedia/go-rtmp"
)

func main() {
	// rtmp://mobliestream.c3tv.com:554/live/goodtv.sdp
	// rtmp://liteavapp.qcloud.com/live/liteavdemoplayerstreamid
	// rtmp://127.0.0.1:1935/live/test
	rtmpUrl := "rtmp://127.0.0.1:1935/live/test111"
	filename := "out.h264"
	var err error
	u, err := url.Parse(rtmpUrl)
	if err != nil {
		panic(err)
	}
	host := u.Host
	if u.Port() == "" {
		host += ":1935"
	}

	//connect to remote rtmp server
	conn, err := net.Dial("tcp4", host)
	if err != nil {
		fmt.Println("连接失败", err)
		return
	} else {
		fmt.Println("连接成功")
	}
	var f1 *os.File
	f1, err = os.Create(filename)
	if err != nil {
		fmt.Println("创建文件失败", err)
		return
	}
	defer f1.Close()
	// var m *mp4.Movmuxer
	// m, err = mp4.CreateMp4Muxer(f1)
	// if err != nil {
	// 	fmt.Println("创建mp4 muxer失败", err)
	// 	return
	// }

	i := 0
	hasVideo := false
	// hasAudeo := false
	// var vtid uint32 = 0
	// var atid uint32 = 0
	c := rtmp.NewRtmpClient(rtmp.WithChunkSize(6000), rtmp.WithComplexHandshake())
	c.OnFrame(func(cid codec.CodecID, pts, dts uint32, frame []byte) {
		if cid == codec.CODECID_VIDEO_H264 || cid == codec.CODECID_VIDEO_H265 {
			if i>=NumberOfFrames{
				return
			}
			if !hasVideo {
				if cid == codec.CODECID_VIDEO_H264 {
					//vtid = m.AddVideoTrack(mp4.MP4_CODEC_H264)
				} else {
					//vtid = m.AddVideoTrack(mp4.MP4_CODEC_H265)
				}
				hasVideo = true
			}
			fmt.Println("数据：", i)
			f1.Write(frame)
			// err := m.Write(vtid, frame, uint64(pts), uint64(dts))
			// if err != nil {
			// 	fmt.Println(err)
			// 	i = NumberOfFrames - 1
			// }
			i++
		} else if cid == codec.CODECID_AUDIO_AAC {
			// if !hasAudeo {
			// 	if cid == codec.CODECID_AUDIO_AAC {
			// 		atid = m.AddVideoTrack(mp4.MP4_CODEC_AAC)
			// 	}

			// 	hasAudeo = true
			// }
			// err := m.Write(atid, frame, uint64(pts), uint64(dts))
			// if err != nil {
			// 	fmt.Println(err)
			// 	i = NumberOfFrames
			// }
			//audioFd.Write(frame)
		}
	})

	//must set output callback
	c.SetOutput(func(b []byte) error {
		_, err := conn.Write(b)
		if err != nil {
			fmt.Println("err", err)
		} else {
			fmt.Println("Write成功")
		}
		return err
	})

	c.Start(rtmpUrl)
	buf := make([]byte, 4096)
	n := 0
	for {
		if i >= NumberOfFrames {
			fmt.Println("1")
			break
		}
		n, err = conn.Read(buf)
		if err != nil {
			fmt.Println("2")
			break
		} else {
			fmt.Println("read成功")
		}
		err = c.Input(buf[:n])
		if err != nil {
			fmt.Println("3")
			break
		} else {
			fmt.Println("Input成功")
		}
	}
	fmt.Println(err)
	// err = m.WriteTrailer()
	// if err != nil {
	// 	fmt.Println("写尾部mp4 muxer失败", err)
	// 	return
	// }
}

const NumberOfFrames = 25 * 20 //30帧 20秒

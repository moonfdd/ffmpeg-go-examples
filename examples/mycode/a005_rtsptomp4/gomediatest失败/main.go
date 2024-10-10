package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/yapingcat/gomedia/go-mp4"
	"github.com/yapingcat/gomedia/go-rtsp"
	"github.com/yapingcat/gomedia/go-rtsp/sdp"
)

var i = 0
var hasVideo = false
var hasAudeo = false
var vtid uint32 = 0
var atid uint32 = 0
var m *mp4.Movmuxer

func main() {
	// rtmp://mobliestream.c3tv.com:554/live/goodtv.sdp
	// rtmp://liteavapp.qcloud.com/live/liteavdemoplayerstreamid
	// rtmp://127.0.0.1:1935/live/test
	rtspUrl := "rtsp://127.0.0.1:5544/live/test111"
	filename := "out.mp4"
	var err error
	u, err := url.Parse(rtspUrl)
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
	m, err = mp4.CreateMp4Muxer(f1)
	if err != nil {
		fmt.Println("创建mp4 muxer失败", err)
		return
	}

	// hasVideo := false
	// hasAudeo := false
	// var vtid uint32 = 0
	// var atid uint32 = 0
	sc := make(chan []byte, 100)
	sess := NewRtspPlaySession(conn)
	go sess.sendInLoop(sc)
	c, err := rtsp.NewRtspClient(rtspUrl, sess)
	if err != nil {
		fmt.Println("NewRtspClient失败", err)
		return
	}
	c.SetOutput(func(b []byte) error {
		if sess.lastError != nil {
			return sess.lastError
		}
		sc <- b
		return nil
	})
	c.Start()

	buf := make([]byte, 4096)
	for {
		if i >= NumberOfFrames {
			fmt.Println("1")
			break
		}
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("2")
			break
		} else {
			fmt.Println("read成功")
		}
		err = c.Input(buf[:n])
		if err != nil {
			fmt.Println("3", err)
			break
		} else {
			fmt.Println("Input成功")
		}
	}
	fmt.Println(err)
	err = m.WriteTrailer()
	if err != nil {
		fmt.Println("写尾部mp4 muxer失败", err)
		return
	}
}

const NumberOfFrames = 25 * 20 //30帧 20秒

type RtspPlaySession struct {
	// videoFile *os.File
	// audioFile *os.File
	// tsFile    *os.File
	timeout   int
	once      sync.Once
	die       chan struct{}
	c         net.Conn
	lastError error
}

func NewRtspPlaySession(c net.Conn) *RtspPlaySession {
	return &RtspPlaySession{die: make(chan struct{}), c: c}
}

func (cli *RtspPlaySession) Destory() {
	cli.once.Do(func() {
		// if cli.videoFile != nil {
		// 	cli.videoFile.Close()
		// }
		// if cli.audioFile != nil {
		// 	cli.audioFile.Close()
		// }
		// if cli.tsFile != nil {
		// 	cli.tsFile.Close()
		// }
		cli.c.Close()
		close(cli.die)
	})
}

func (cli *RtspPlaySession) HandleOption(client *rtsp.RtspClient, res rtsp.RtspResponse, public []string) error {
	fmt.Println("rtsp server public ", public)
	return nil
}

func (cli *RtspPlaySession) HandleDescribe(client *rtsp.RtspClient, res rtsp.RtspResponse, sdp *sdp.Sdp, tracks map[string]*rtsp.RtspTrack) error {
	fmt.Println("handle describe ", res.StatusCode, res.Reason)
	for k, t := range tracks {
		if t == nil {
			continue
		}
		fmt.Println("Got ", k, " track")
		if t.Codec.Cid == rtsp.RTSP_CODEC_H264 {
			if i >= NumberOfFrames {
				return nil
			}
			if !hasVideo {
				vtid = m.AddVideoTrack(mp4.MP4_CODEC_H264)
				hasVideo = true
			}

			fmt.Println("数据：", i)
			t.OnSample(func(sample rtsp.RtspSample) {
				//fmt.Println("Got H264 Frame size:", len(sample.Sample), " timestamp:", sample.Timestamp)
				// cli.videoFile.Write(sample.Sample)
				err := m.Write(vtid, sample.Sample, uint64(sample.Timestamp), uint64(sample.Timestamp))
				if err != nil {
					fmt.Println(err)
					i = NumberOfFrames
				}
				i++
			})
			// if cli.videoFile == nil {
			// 	cli.videoFile, _ = os.OpenFile("video.h264", os.O_CREATE|os.O_RDWR, 0666)
			// }
			// t.OnSample(func(sample rtsp.RtspSample) {
			// 	//fmt.Println("Got H264 Frame size:", len(sample.Sample), " timestamp:", sample.Timestamp)
			// 	cli.videoFile.Write(sample.Sample)
			// })
		} else if t.Codec.Cid == rtsp.RTSP_CODEC_AAC {
			if i >= NumberOfFrames {
				return nil
			}
			if !hasAudeo {
				atid = m.AddVideoTrack(mp4.MP4_CODEC_AAC)
				hasAudeo = true
			}
			t.OnSample(func(sample rtsp.RtspSample) {
				//fmt.Println("Got H264 Frame size:", len(sample.Sample), " timestamp:", sample.Timestamp)
				// cli.videoFile.Write(sample.Sample)
				err := m.Write(atid, sample.Sample, uint64(sample.Timestamp), uint64(sample.Timestamp))
				if err != nil {
					fmt.Println(err)
					i = NumberOfFrames
				}
			})
			// if cli.audioFile == nil {
			// 	cli.audioFile, _ = os.OpenFile("audio.aac", os.O_CREATE|os.O_RDWR, 0666)
			// }
			// t.OnSample(func(sample rtsp.RtspSample) {
			// 	//fmt.Println("Got AAC Frame size:", len(sample.Sample), " timestamp:", sample.Timestamp)
			// 	cli.audioFile.Write(sample.Sample)
			// })
		} else if t.Codec.Cid == rtsp.RTSP_CODEC_TS {
			// if cli.tsFile == nil {
			// 	cli.tsFile, _ = os.OpenFile("mp2t.ts", os.O_CREATE|os.O_RDWR, 0666)
			// }
			// t.OnSample(func(sample rtsp.RtspSample) {
			// 	cli.tsFile.Write(sample.Sample)
			// })
		}
	}
	return nil
}

func (cli *RtspPlaySession) HandleSetup(client *rtsp.RtspClient, res rtsp.RtspResponse, track *rtsp.RtspTrack, tracks map[string]*rtsp.RtspTrack, sessionId string, timeout int) error {
	fmt.Println("HandleSetup sessionid:", sessionId, " timeout:", timeout)
	cli.timeout = timeout
	return nil
}

func (cli *RtspPlaySession) HandleAnnounce(client *rtsp.RtspClient, res rtsp.RtspResponse) error {
	return nil
}

func (cli *RtspPlaySession) HandlePlay(client *rtsp.RtspClient, res rtsp.RtspResponse, timeRange *rtsp.RangeTime, info *rtsp.RtpInfo) error {
	if res.StatusCode != 200 {
		fmt.Println("play failed ", res.StatusCode, res.Reason)
		return nil
	}
	go func() {
		//rtsp keepalive
		to := time.NewTicker(time.Duration(cli.timeout/2) * time.Second)
		defer to.Stop()
		for {
			select {
			case <-to.C:
				client.KeepAlive(rtsp.OPTIONS)
			case <-cli.die:
				return
			}
		}
	}()
	return nil
}

func (cli *RtspPlaySession) HandlePause(client *rtsp.RtspClient, res rtsp.RtspResponse) error {
	return nil
}

func (cli *RtspPlaySession) HandleTeardown(client *rtsp.RtspClient, res rtsp.RtspResponse) error {
	return nil
}

func (cli *RtspPlaySession) HandleGetParameter(client *rtsp.RtspClient, res rtsp.RtspResponse) error {
	return nil
}

func (cli *RtspPlaySession) HandleSetParameter(client *rtsp.RtspClient, res rtsp.RtspResponse) error {
	return nil
}

func (cli *RtspPlaySession) HandleRedirect(client *rtsp.RtspClient, req rtsp.RtspRequest, location string, timeRange *rtsp.RangeTime) error {
	return nil
}

func (cli *RtspPlaySession) HandleRecord(client *rtsp.RtspClient, res rtsp.RtspResponse, timeRange *rtsp.RangeTime, info *rtsp.RtpInfo) error {
	return nil
}

func (cli *RtspPlaySession) HandleRequest(client *rtsp.RtspClient, req rtsp.RtspRequest) error {
	return nil
}

func (cli *RtspPlaySession) sendInLoop(sendChan chan []byte) {
	for {
		select {
		case b := <-sendChan:
			_, err := cli.c.Write(b)
			if err != nil {
				cli.Destory()
				cli.lastError = err
				fmt.Println("quit send in loop")
				return
			}

		case <-cli.die:
			fmt.Println("quit send in loop")
			return
		}
	}
}

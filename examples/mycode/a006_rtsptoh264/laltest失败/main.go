package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/remux"
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/rtsp"
	"github.com/q191201771/lal/pkg/sdp"
)

func main() {
	url := "rtsp://127.0.0.1:5544/live/test111"
	filename := "out.h264"

	var err error
	// var w httpflv.FlvFileWriter
	// err = w.Open(filename)
	// if err != nil {
	// 	fmt.Println("flv文件open失败", err)
	// 	return
	// }
	// defer w.Dispose()
	// err = w.WriteRaw(httpflv.FlvHeader)
	// if err != nil {
	// 	fmt.Println("flv文件WriteRaw失败", err)
	// 	return
	// }
	var w *os.File
	w, err = os.Create(filename)
	if err != nil {
		fmt.Println("创建文件失败", err)
		return
	}
	defer w.Close()

	i := 0
	var overTcp bool
	remuxer = remux.NewAvPacket2RtmpRemuxer().WithOnRtmpMsg(func(msg base.RtmpMsg) {
		if i >= NumberOfFrames {
			os.Exit(0)
			return
		}
		tag := remux.RtmpMsg2FlvTag(msg)
		if tag.Header.Type == 8 {
			return
		}
		if tag.Header.Type == 9 {
			i++
		}
		d, _ := json.MarshalIndent(tag, "", "  ")
		_ = d
		// fmt.Println(string(d))
		// err := w.WriteTag(*tag)
		// if err != nil {
		// 	fmt.Println("flv文件WriteTag失败", err)
		// }
		w.Write([]byte{0, 0, 0, 1})
		w.Write(msg.Payload)
		// nazalog.Assert(nil, err)
	})
	var observer Observer
	pullSession := rtsp.NewPullSession(&observer, func(option *rtsp.PullSessionOption) {
		option.PullTimeoutMs = 10000
		option.OverTcp = overTcp != false
	})

	err = pullSession.Pull(url)
	go func() {
		for {
			pullSession.UpdateStat(1)
			// nazalog.Debugf("stat. pull=%+v", pullSession.GetStat())
			time.Sleep(1 * time.Second)
		}
	}()
	err = <-pullSession.WaitChan()
	fmt.Printf("< pullSession.Wait(). err=%+v", err)

}

// const NumberOfFrames = 30 * 20 //30帧 20秒
const NumberOfFrames = 25 * 20 //30帧 20秒
var remuxer *remux.AvPacket2RtmpRemuxer
var dump *base.DumpFile

type Observer struct{}

func (o *Observer) OnSdp(sdpCtx sdp.LogicContext) {
	// nazalog.Debugf("OnSdp %+v", sdpCtx)
	if dump != nil {
		dump.WriteWithType(sdpCtx.RawSdp, base.DumpTypeRtspSdpData)
	}
	remuxer.OnSdp(sdpCtx)
}

func (o *Observer) OnRtpPacket(pkt rtprtcp.RtpPacket) {
	if dump != nil {
		dump.WriteWithType(pkt.Raw, base.DumpTypeRtspRtpData)
	}
	remuxer.OnRtpPacket(pkt)
}

func (o *Observer) OnAvPacket(pkt base.AvPacket) {
	//nazalog.Debugf("OnAvPacket %+v", pkt.DebugString())
	remuxer.OnAvPacket(pkt)
}

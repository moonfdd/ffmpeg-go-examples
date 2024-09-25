package videohelper

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"github.com/moonfdd/ffmpeg-go/ffcommon"
	"github.com/moonfdd/ffmpeg-go/libavcodec"
	"github.com/moonfdd/ffmpeg-go/libavdevice"
	"github.com/moonfdd/ffmpeg-go/libavformat"
	"github.com/moonfdd/ffmpeg-go/libavutil"
	"github.com/moonfdd/ffmpeg-go/libswscale"
	sdl "github.com/moonfdd/sdl2-go/sdl2"
	"github.com/moonfdd/sdl2-go/sdlcommon"
	"github.com/ying32/govcl/vcl"
)

// 只能调用一次，程序开始时调用
func Init() (isSuc bool) {
	e := os.Getenv("Path")
	if !strings.HasSuffix(e, ";") {
		e += ";"
	}
	os.Setenv("Path", os.Getenv("Path")+";./lib/windows/ffmpeg;./lib/windows/vlc;")
	ffcommon.SetAvutilPath("./lib/windows/ffmpeg/avutil-56.dll")
	ffcommon.SetAvcodecPath("./lib/windows/ffmpeg/avcodec-58.dll")
	ffcommon.SetAvdevicePath("./lib/windows/ffmpeg/avdevice-58.dll")
	ffcommon.SetAvfilterPath("./lib/windows/ffmpeg/avfilter-56.dll")
	ffcommon.SetAvformatPath("./lib/windows/ffmpeg/avformat-58.dll")
	ffcommon.SetAvpostprocPath("./lib/windows/ffmpeg/postproc-55.dll")
	ffcommon.SetAvswresamplePath("./lib/windows/ffmpeg/swresample-3.dll")
	ffcommon.SetAvswscalePath("./lib/windows/ffmpeg/swscale-5.dll")
	// sdlcommon.SetSDL2Path("./lib/windows/sdl/SDL2.0.16.dll")
	sdlcommon.SetSDL2Path("./lib/windows/sdl/SDL2.30.7.dll")
	libavdevice.AvdeviceRegisterAll()
	libavformat.AvformatNetworkInit()
	if sdl.SDL_Init(sdl.SDL_INIT_VIDEO) != 0 {
		fmt.Printf("Could not initialize SDL - %s\n", sdl.SDL_GetError())
		return
	}
	isSuc = true
	return
}

type Playinfo struct {
	VideoAddr string
	Hwnd      uintptr

	IsStop bool
	//loopShow最后发送 stop里接收
	FinishCh chan struct{}
}

// 播放
func Play(videoAddr string, hwnd uintptr) *Playinfo {
	pi := &Playinfo{VideoAddr: videoAddr, Hwnd: hwnd, FinishCh: make(chan struct{}, 1)}
	go loopShow(pi)
	return pi
}
func loopShow(pi *Playinfo) {
	var i int
	for !pi.IsStop {
		fmt.Println("运行", i)
		runShow(pi)
		i++
	}
	fmt.Println("停止", i)
	pi.FinishCh <- struct{}{}
}

func runShow(pi *Playinfo) {
	var pFormatCtx *libavformat.AVFormatContext
	var i, videoindex ffcommon.FInt
	var pCodecCtx *libavcodec.AVCodecContext
	var pCodec *libavcodec.AVCodec
	var ifmt *libavformat.AVInputFormat

	// libavformat.AvRegisterAll()
	// libavformat.AvformatNetworkInit()
	// pFormatCtx = libavformat.AvformatAllocContext()
	// defer pFormatCtx.AvformatFreeContext()
	// ifmt = libavformat.AvFindInputFormat("dshow")
	var options *libavutil.AVDictionary
	libavutil.AvDictSet(&options, "stimeout", "5000000", 0)
	libavutil.AvDictSet(&options, "rw_timeout", "1000000", 0)
	libavutil.AvDictSet(&options, "reconnect", "0", 0)
	libavutil.AvDictSet(&options, "reconnect_at_eof", "0", 0)
	// libavutil.AvDictSetInt(&options, "timeout", 10000000, 0) //连不上
	// libavutil.AvDictSet(&options, "rtbufsize", "100000000", 0)
	fmt.Println("AvformatOpenInput前")
	defer libavutil.AvDictFree(&options)
	if libavformat.AvformatOpenInput(&pFormatCtx, pi.VideoAddr, ifmt, &options) < 0 {
		// vcl.ShowMessage("Cannot open camera.\n")
		fmt.Println("AvformatOpenInput失败")
		// libavutil.AvDictFree(&options)
		return
	}
	// defer pFormatCtx.AvformatFreeContext()
	defer libavformat.AvformatCloseInput(&pFormatCtx)
	fmt.Println("AvformatOpenInput后")
	// pFormatCtx.InterruptCallback.Callback
	// libavutil.AvDictFree(&options)

	if pFormatCtx.AvformatFindStreamInfo(&options) < 0 {
		// vcl.ShowMessage("Couldn't find stream information.")
		return
	}
	videoindex = -1
	for i = 0; i < int32(pFormatCtx.NbStreams); i++ {
		if pFormatCtx.GetStream(uint32(i)).Codec.CodecType == libavutil.AVMEDIA_TYPE_VIDEO {
			videoindex = i
			break
		}
	}
	if videoindex == -1 {
		// vcl.ShowMessage("Didn't find a video stream.\n")
		return
	}
	pCodecCtxPara := pFormatCtx.GetStream(uint32(videoindex)).Codecpar
	pCodec = libavcodec.AvcodecFindDecoder(pCodecCtxPara.CodecId)
	if pCodec == nil {
		// vcl.ShowMessage("Codec not found.\n")
		return
	}

	pCodecCtx = pCodec.AvcodecAllocContext3()
	if pCodecCtx == nil {
		// vcl.ShowMessage("Cannot alloc valid decode codec context.\n")
		return
	}
	defer pCodecCtx.AvcodecClose()

	if pCodecCtx.AvcodecParametersToContext(pCodecCtxPara) < 0 {
		// vcl.ShowMessage("Cannot initialize parameters.\n")
		return
	}

	if pCodecCtx.AvcodecOpen2(pCodec, nil) < 0 {
		// vcl.ShowMessage("Could not open codec.\n")
		return
	}

	var pFrame, pFrameYUV *libavutil.AVFrame
	pFrame = libavutil.AvFrameAlloc()
	defer libavutil.AvFree(uintptr(unsafe.Pointer(pFrame)))
	pFrameYUV = libavutil.AvFrameAlloc()
	defer libavutil.AvFree(uintptr(unsafe.Pointer(pFrameYUV)))
	//unsigned char *out_buffer=(unsigned char *)av_malloc(avpicture_get_size(AV_PIX_FMT_YUV420P, pCodecCtx->width, pCodecCtx->height));
	//avpicture_fill((AVPicture *)pFrameYUV, out_buffer, AV_PIX_FMT_YUV420P, pCodecCtx->width, pCodecCtx->height);
	out_buffer := (*byte)(unsafe.Pointer(libavutil.AvMalloc(uint64(libavcodec.AvpictureGetSize(int32(libavutil.AV_PIX_FMT_YUV420P), pCodecCtx.Width, pCodecCtx.Height)))))
	((*libavcodec.AVPicture)(unsafe.Pointer(pFrameYUV))).AvpictureFill(out_buffer, libavutil.AV_PIX_FMT_YUV420P, pCodecCtx.Width, pCodecCtx.Height)
	defer libavutil.AvFree(uintptr(unsafe.Pointer(out_buffer)))
	//SDL----------------------------
	// if sdl.SDL_Init(sdl.SDL_INIT_VIDEO|sdl.SDL_INIT_AUDIO|sdl.SDL_INIT_TIMER) != 0 {
	// if sdl.SDL_Init(sdl.SDL_INIT_VIDEO) != 0 {
	// 	vcl.ShowMessage(fmt.Sprintf("Could not initialize SDL - %s\n", sdl.SDL_GetError()))
	// }
	// var screen_w, screen_h ffcommon.FInt = 640, 360
	// var mode *sdl.SDL_DisplayMode = new(sdl.SDL_DisplayMode)
	// if sdl.SDL_GetCurrentDisplayMode(0, mode) != 0 {
	// 	fmt.Printf("SDL: could not get current display mode - exiting:%s\n", sdl.SDL_GetError())
	// 	return -1
	// }
	//Half of the Desktop's width and height.
	// screen_w = pCodecCtx.Width
	// screen_h = pCodecCtx.Height
	// screen_w = f.Panel1.Width()
	// screen_h = f.Panel1.Height()
	window := sdl.SDL_CreateWindowFrom(pi.Hwnd)
	if window == nil {
		// vcl.ShowMessage(fmt.Sprintf("SDL: could not create window - exiting:%s\n", sdl.SDL_GetError()))
		return
	}
	defer window.SDL_DestroyWindow()
	// defer window.SDL_DestroyWindow()
	// window.Flags = sdl.SDL_WINDOW_FOREIGN

	renderer := window.SDL_CreateRenderer(-1, 0)
	if renderer == nil {
		// vcl.ShowMessage(fmt.Sprintf("SDL: could not create renderer - exiting:%s\n", sdl.SDL_GetError()))
		return
	}
	defer renderer.SDL_DestroyRenderer()
	texture := renderer.SDL_CreateTexture(sdl.SDL_PIXELFORMAT_YV12,
		sdl.SDL_TEXTUREACCESS_STREAMING,
		pCodecCtx.Width,
		pCodecCtx.Height)
	defer texture.SDL_DestroyTexture()
	// var rect sdl.SDL_Rect
	// rect.X = 0
	// rect.Y = 0
	// rect.W = screen_w
	// rect.H = screen_h
	// var rect2 sdl.SDL_Rect
	// rect2.X = 0
	// rect2.Y = 0
	// rect2.W = pCodecCtx.Width
	// rect2.H = pCodecCtx.Height
	packet := &libavcodec.AVPacket{}
	var img_convert_ctx *libswscale.SwsContext
	var ret int32
	img_convert_ctx = libswscale.SwsGetContext(pCodecCtx.Width, pCodecCtx.Height, pCodecCtx.PixFmt, pCodecCtx.Width, pCodecCtx.Height, libavutil.AV_PIX_FMT_YUV420P, libswscale.SWS_BICUBIC, nil, nil, nil)
	defer img_convert_ctx.SwsFreeContext()
	var ii int
	// pFormatCtx.InterruptCallback.Callback = ffcommon.NewCallback(InterruptFouction)
	// now := time.Now()
	// pFormatCtx.InterruptCallback.Opaque = uintptr(unsafe.Pointer(&now))
	for !pi.IsStop {
		fmt.Println("AvReadFrame", ii)
		ii++
		// now = time.Now()
		if pFormatCtx.AvReadFrame(packet) >= 0 {
			fmt.Println("AvReadFrame end>0", ii)
			if int32(packet.StreamIndex) == videoindex {
				if pCodecCtx.AvcodecSendPacket(packet) < 0 {
					packet.AvPacketUnref()
					// vcl.ShowMessage(fmt.Sprintf("pCodecCtx.AvcodecSendPacket(packet) < 0\n"))
					return

				}
				ret = pCodecCtx.AvcodecReceiveFrame(pFrame)

				if ret < 0 {
					packet.AvFreePacket()
					continue
					vcl.ShowMessage(fmt.Sprintf("Decode Error.\n"))
					return
				}
				if ret >= 0 {
					// if got_picture != 0 {
					img_convert_ctx.SwsScale((**byte)(unsafe.Pointer(&pFrame.Data)), (*int32)(unsafe.Pointer(&pFrame.Linesize)), 0, uint32(pCodecCtx.Height), (**byte)(unsafe.Pointer(&pFrameYUV.Data)), (*int32)(unsafe.Pointer(&pFrameYUV.Linesize)))

					texture.SDL_UpdateYUVTexture(nil,
						pFrameYUV.Data[0], pFrameYUV.Linesize[0],
						pFrameYUV.Data[1], pFrameYUV.Linesize[1],
						pFrameYUV.Data[2], pFrameYUV.Linesize[2])

					renderer.SDL_RenderClear()
					renderer.SDL_RenderCopy(texture, nil, nil)
					renderer.SDL_RenderPresent()
				}
			}
			packet.AvFreePacket()
		} else {
			fmt.Println("AvReadFrame end", ii)
			//vcl.ShowMessage(fmt.Sprintf("pFormatCtx.AvReadFrame(packet) < 0\n"))
			return
		}
	}
}

// func InterruptFouction(now uintptr) uintptr {
// 	return 1
// 	// fmt.Println("InterruptFouction")
// 	// if now == nil {
// 	// 	return 0
// 	// }
// 	// fmt.Println("入now = ", now)
// 	// fmt.Println("now = ", time.Now())
// 	// return 0
// }

// 停止
func Stop(pi *Playinfo) {
	pi.IsStop = true
	<-pi.FinishCh
}

// 只能调用一次，程序结束时调用
func Dispose() {
	sdl.SDL_Quit()
	libavformat.AvformatNetworkDeinit()
}

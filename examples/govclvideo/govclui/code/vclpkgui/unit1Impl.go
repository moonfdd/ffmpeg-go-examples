package vclpkgui

import (
	"fmt"
	"os"
	"unsafe"

	"github.com/deepch/vdk/av"
	"github.com/deepch/vdk/codec/h264parser"
	"github.com/deepch/vdk/format/rtmp"
	"github.com/moonfdd/ffmpeg-go-examples/examples/govclvideo/govclui/code/videohelper"
	"github.com/moonfdd/ffmpeg-go/ffcommon"
	"github.com/moonfdd/ffmpeg-go/libavcodec"
	"github.com/moonfdd/ffmpeg-go/libavdevice"
	"github.com/moonfdd/ffmpeg-go/libavformat"
	"github.com/moonfdd/ffmpeg-go/libavutil"
	"github.com/moonfdd/ffmpeg-go/libswscale"
	sdl "github.com/moonfdd/sdl2-go/sdl2"
	"github.com/moonfdd/sdl2-go/sdlcommon"
	"github.com/ying32/govcl/pkgs/libvlc"
	"github.com/ying32/govcl/vcl"
	"github.com/ying32/govcl/vcl/rtl"
)

// ::private::
type TForm1Fields struct {
}

// 桌面显示
func (f *TForm1) OnButton1Click(sender vcl.IObject) {
	go func() {
		var pFormatCtx *libavformat.AVFormatContext
		var i, videoindex ffcommon.FInt
		var pCodecCtx *libavcodec.AVCodecContext
		var pCodec *libavcodec.AVCodec
		var ifmt *libavformat.AVInputFormat
		var options *libavutil.AVDictionary
		pFormatCtx = libavformat.AvformatAllocContext()
		defer libavformat.AvformatCloseInput(&pFormatCtx)
		ifmt = libavformat.AvFindInputFormat("gdigrab")
		// ifmt = libavformat.AvFindInputFormat("dshow")
		if libavformat.AvformatOpenInput(&pFormatCtx, "desktop", ifmt, &options) != 0 {
			vcl.ShowMessage("Couldn't open input stream2.\n")
			return
		}
		if pFormatCtx.AvformatFindStreamInfo(nil) < 0 {
			vcl.ShowMessage("Couldn't find stream information.")
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
			vcl.ShowMessage("Didn't find a video stream.\n")
			return
		}
		pCodecCtx = pFormatCtx.GetStream(uint32(videoindex)).Codec
		defer pCodecCtx.AvcodecClose()
		pCodec = libavcodec.AvcodecFindDecoder(pCodecCtx.CodecId)
		if pCodec == nil {
			vcl.ShowMessage("Codec not found.\n")
			return
		}
		if pCodecCtx.AvcodecOpen2(pCodec, nil) < 0 {
			vcl.ShowMessage("Could not open codec.\n")
			return
		}

		var pFrame, pFrameYUV *libavutil.AVFrame
		pFrame = libavutil.AvFrameAlloc()
		defer libavutil.AvFree(uintptr(unsafe.Pointer(pFrame)))
		pFrameYUV = libavutil.AvFrameAlloc()
		defer libavutil.AvFree(uintptr(unsafe.Pointer(pFrameYUV)))
		//unsigned char *out_buffer=(unsigned char *)av_malloc(avpicture_get_size(AV_PIX_FMT_YUV420P, pCodecCtx->width, pCodecCtx->height));
		//avpicture_fill((AVPicture *)pFrameYUV, out_buffer, AV_PIX_FMT_YUV420P, pCodecCtx->width, pCodecCtx->height);
		out_buffer := (*byte)(unsafe.Pointer(libavutil.AvMalloc(uint64(libavcodec.AvpictureGetSize(libavutil.AV_PIX_FMT_YUV420P, pCodecCtx.Width, pCodecCtx.Height)))))
		defer libavutil.AvFree(uintptr(unsafe.Pointer(out_buffer)))
		((*libavcodec.AVPicture)(unsafe.Pointer(pFrameYUV))).AvpictureFill(out_buffer, libavutil.AV_PIX_FMT_YUV420P, pCodecCtx.Width, pCodecCtx.Height)
		var screen_w, screen_h ffcommon.FInt

		var mode *sdl.SDL_DisplayMode = new(sdl.SDL_DisplayMode)
		if sdl.SDL_GetCurrentDisplayMode(0, mode) != 0 {
			vcl.ShowMessage(fmt.Sprintf("SDL: could not get current display mode - exiting:%s\n", sdl.SDL_GetError()))
			return
		}

		// vcl.ShowMessage(fmt.Sprint(mode.W,mode.H))
		screen_w = f.Panel1.Width()
		screen_h = f.Panel1.Height()
		window := sdl.SDL_CreateWindowFrom(f.Panel1.Handle())
		if window == nil {
			vcl.ShowMessage(fmt.Sprintf("SDL: could not create window - exiting:%s\n", sdl.SDL_GetError()))
			return
		}
		defer window.SDL_DestroyWindow()
		renderer := window.SDL_CreateRenderer(-1, 0)
		if renderer == nil {
			vcl.ShowMessage(fmt.Sprintf("SDL: could not create renderer - exiting:%s\n", sdl.SDL_GetError()))
			return
		}
		defer renderer.SDL_DestroyRenderer()
		texture := renderer.SDL_CreateTexture(sdl.SDL_PIXELFORMAT_YV12,
			sdl.SDL_TEXTUREACCESS_STREAMING,
			pCodecCtx.Width,
			pCodecCtx.Height)
		defer texture.SDL_DestroyTexture()
		window.SDL_ShowWindow()
		var rect sdl.SDL_Rect
		rect.X = 0
		rect.Y = 0
		rect.W = screen_w
		rect.H = screen_h
		var rect2 sdl.SDL_Rect
		rect2.X = 0
		rect2.Y = 0
		rect2.W = mode.W
		rect2.H = mode.H
		packet := &libavcodec.AVPacket{}
		var img_convert_ctx *libswscale.SwsContext
		var ret ffcommon.FInt
		img_convert_ctx = libswscale.SwsGetContext(pCodecCtx.Width, pCodecCtx.Height, pCodecCtx.PixFmt, pCodecCtx.Width, pCodecCtx.Height, libavutil.AV_PIX_FMT_YUV420P, libswscale.SWS_BICUBIC, nil, nil, nil)
		defer img_convert_ctx.SwsFreeContext()
		for {
			if pFormatCtx.AvReadFrame(packet) >= 0 {
				if int32(packet.StreamIndex) == videoindex {
					if pCodecCtx.AvcodecSendPacket(packet) < 0 {
						packet.AvPacketUnref()
						continue

					}
					ret = pCodecCtx.AvcodecReceiveFrame(pFrame)
					if ret < 0 {
						fmt.Printf("Decode Error.\n")
						return
					}
					if ret >= 0 {
						// if got_picture != 0 {
						img_convert_ctx.SwsScale((**byte)(unsafe.Pointer(&pFrame.Data)), (*int32)(unsafe.Pointer(&pFrame.Linesize)), 0, uint32(pCodecCtx.Height), (**byte)(unsafe.Pointer(&pFrameYUV.Data)), (*int32)(unsafe.Pointer(&pFrameYUV.Linesize)))
						texture.SDL_UpdateYUVTexture(&rect2,
							pFrameYUV.Data[0], pFrameYUV.Linesize[0],
							pFrameYUV.Data[1], pFrameYUV.Linesize[1],
							pFrameYUV.Data[2], pFrameYUV.Linesize[2])

						renderer.SDL_RenderClear()
						renderer.SDL_RenderCopy(texture, nil, &rect)
						renderer.SDL_RenderPresent()

					}
				}
				packet.AvPacketUnref()
			}
		}

	}()
}

// 桌面停止
func (f *TForm1) OnButton2Click(sender vcl.IObject) {

}

// 初始化
func (f *TForm1) OnButton3Click(sender vcl.IObject) {
	videohelper.Init()
	return
	os.Setenv("VLC_PLUGIN_PATH", rtl.ExtractFilePath(vcl.Application.ExeName())+"/lib/windows/vlc/plugins/")
	// os.Setenv("VLC_PLUGIN_PATH", rtl.ExtractFilePath(vcl.Application.ExeName())+"/plugins/")
	os.Setenv("Path", os.Getenv("Path")+";./lib/windows/ffmpeg;./lib/windows/vlc;")
	ffcommon.SetAvutilPath("./lib/windows/ffmpeg/avutil-56.dll")
	ffcommon.SetAvcodecPath("./lib/windows/ffmpeg/avcodec-58.dll")
	ffcommon.SetAvdevicePath("./lib/windows/ffmpeg/avdevice-58.dll")
	ffcommon.SetAvfilterPath("./lib/windows/ffmpeg/avfilter-56.dll")
	ffcommon.SetAvformatPath("./lib/windows/ffmpeg/avformat-58.dll")
	ffcommon.SetAvpostprocPath("./lib/windows/ffmpeg/postproc-55.dll")
	ffcommon.SetAvswresamplePath("./lib/windows/ffmpeg/swresample-3.dll")
	ffcommon.SetAvswscalePath("./lib/windows/ffmpeg/swscale-5.dll")
	sdlcommon.SetSDL2Path("./lib/windows/sdl/SDL2.0.16.dll")

	// libavformat.AvRegisterAll()
	// libavformat.AvformatNetworkInit()

	libavdevice.AvdeviceRegisterAll()
	if sdl.SDL_Init(sdl.SDL_INIT_VIDEO) != 0 {
		fmt.Printf("Could not initialize SDL - %s\n", sdl.SDL_GetError())
		return
	}
	vcl.ShowMessage("初始化成功")
}

// 摄像头显示
func (f *TForm1) OnButton4Click(sender vcl.IObject) {
	go func() {
		var pFormatCtx *libavformat.AVFormatContext
		var i, videoindex ffcommon.FInt
		var pCodecCtx *libavcodec.AVCodecContext
		var pCodec *libavcodec.AVCodec
		var ifmt *libavformat.AVInputFormat

		libavformat.AvRegisterAll()
		libavformat.AvformatNetworkInit()
		pFormatCtx = libavformat.AvformatAllocContext()
		defer libavformat.AvformatCloseInput(&pFormatCtx)
		ifmt = libavformat.AvFindInputFormat("dshow")
		var options *libavutil.AVDictionary
		// libavutil.AvDictSet(&options, "probesize", "100000000", 0)
		// libavutil.AvDictSet(&options, "rtbufsize", "100000000", 0)
		if libavformat.AvformatOpenInput(&pFormatCtx, "video=Full HD webcam", ifmt, &options) < 0 {
			vcl.ShowMessage("Cannot open camera.\n")
			return
		}

		if pFormatCtx.AvformatFindStreamInfo(nil) < 0 {
			vcl.ShowMessage("Couldn't find stream information.")
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
			vcl.ShowMessage("Didn't find a video stream.\n")
			return
		}
		pCodecCtxPara := pFormatCtx.GetStream(uint32(videoindex)).Codecpar
		pCodec = libavcodec.AvcodecFindDecoder(pCodecCtxPara.CodecId)
		if pCodec == nil {
			vcl.ShowMessage("Codec not found.\n")
			return
		}

		pCodecCtx = pCodec.AvcodecAllocContext3()
		if pCodecCtx == nil {
			vcl.ShowMessage("Cannot alloc valid decode codec context.\n")
			return
		}
		defer pCodecCtx.AvcodecClose()

		if pCodecCtx.AvcodecParametersToContext(pCodecCtxPara) < 0 {
			vcl.ShowMessage("Cannot initialize parameters.\n")
			return
		}

		if pCodecCtx.AvcodecOpen2(pCodec, nil) < 0 {
			vcl.ShowMessage("Could not open codec.\n")
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
		var screen_w, screen_h ffcommon.FInt = 640, 360
		// var mode *sdl.SDL_DisplayMode = new(sdl.SDL_DisplayMode)
		// if sdl.SDL_GetCurrentDisplayMode(0, mode) != 0 {
		// 	fmt.Printf("SDL: could not get current display mode - exiting:%s\n", sdl.SDL_GetError())
		// 	return -1
		// }
		//Half of the Desktop's width and height.
		// screen_w = pCodecCtx.Width
		// screen_h = pCodecCtx.Height
		screen_w = f.Panel1.Width()
		screen_h = f.Panel1.Height()
		window := sdl.SDL_CreateWindowFrom(f.Panel2.Handle())
		if window == nil {
			vcl.ShowMessage(fmt.Sprintf("SDL: could not create window - exiting:%s\n", sdl.SDL_GetError()))
			return
		}
		defer window.SDL_DestroyWindow()
		renderer := window.SDL_CreateRenderer(-1, 0)
		if renderer == nil {
			vcl.ShowMessage(fmt.Sprintf("SDL: could not create renderer - exiting:%s\n", sdl.SDL_GetError()))
			return
		}
		defer renderer.SDL_DestroyRenderer()
		texture := renderer.SDL_CreateTexture(sdl.SDL_PIXELFORMAT_YV12,
			sdl.SDL_TEXTUREACCESS_STREAMING,
			pCodecCtx.Width,
			pCodecCtx.Height)
		defer texture.SDL_DestroyTexture()
		var rect sdl.SDL_Rect
		rect.X = 0
		rect.Y = 0
		rect.W = screen_w
		rect.H = screen_h
		var rect2 sdl.SDL_Rect
		rect2.X = 0
		rect2.Y = 0
		rect2.W = pCodecCtx.Width
		rect2.H = pCodecCtx.Height
		packet := &libavcodec.AVPacket{}
		var img_convert_ctx *libswscale.SwsContext
		var ret int32
		img_convert_ctx = libswscale.SwsGetContext(pCodecCtx.Width, pCodecCtx.Height, pCodecCtx.PixFmt, pCodecCtx.Width, pCodecCtx.Height, libavutil.AV_PIX_FMT_YUV420P, libswscale.SWS_BICUBIC, nil, nil, nil)
		defer img_convert_ctx.SwsFreeContext()
		for {
			if pFormatCtx.AvReadFrame(packet) >= 0 {
				if int32(packet.StreamIndex) == videoindex {
					if pCodecCtx.AvcodecSendPacket(packet) < 0 {
						packet.AvPacketUnref()
						vcl.ShowMessage(fmt.Sprintf("pCodecCtx.AvcodecSendPacket(packet) < 0\n"))
						return

					}
					ret = pCodecCtx.AvcodecReceiveFrame(pFrame)
					if ret < 0 {
						vcl.ShowMessage(fmt.Sprintf("Decode Error.\n"))
						return
					}
					if ret >= 0 {
						// if got_picture != 0 {
						img_convert_ctx.SwsScale((**byte)(unsafe.Pointer(&pFrame.Data)), (*int32)(unsafe.Pointer(&pFrame.Linesize)), 0, uint32(pCodecCtx.Height), (**byte)(unsafe.Pointer(&pFrameYUV.Data)), (*int32)(unsafe.Pointer(&pFrameYUV.Linesize)))

						texture.SDL_UpdateYUVTexture(&rect2,
							pFrameYUV.Data[0], pFrameYUV.Linesize[0],
							pFrameYUV.Data[1], pFrameYUV.Linesize[1],
							pFrameYUV.Data[2], pFrameYUV.Linesize[2])

						renderer.SDL_RenderClear()
						renderer.SDL_RenderCopy(texture, nil, &rect)
						renderer.SDL_RenderPresent()

					}
				}

				packet.AvFreePacket()
			} else {
				vcl.ShowMessage(fmt.Sprintf("pFormatCtx.AvReadFrame(packet) < 0\n"))
				return
			}
		}
	}()
}

// 摄像头停止
func (f *TForm1) OnButton5Click(sender vcl.IObject) {

}

// https://www.cnblogs.com/kn-zheng/p/17411093.html
// rtmp显示
func (f *TForm1) OnButton6Click(sender vcl.IObject) {
	// f.Edit1.SetText("http://www.w3school.com.cn/i/movie.mp4")
	// f.Edit1.SetText("rtmp://liteavapp.qcloud.com/live/liteavdemoplayerstreamid")
	go func() {
		var pFormatCtx *libavformat.AVFormatContext
		var i, videoindex ffcommon.FInt
		var pCodecCtx *libavcodec.AVCodecContext
		var pCodec *libavcodec.AVCodec
		var ifmt *libavformat.AVInputFormat

		libavformat.AvRegisterAll()
		libavformat.AvformatNetworkInit()
		pFormatCtx = libavformat.AvformatAllocContext()
		defer libavformat.AvformatCloseInput(&pFormatCtx)
		// ifmt = libavformat.AvFindInputFormat("dshow")
		var options *libavutil.AVDictionary
		// libavutil.AvDictSet(&options, "probesize", "100000000", 0)
		// libavutil.AvDictSet(&options, "rtbufsize", "100000000", 0)
		if libavformat.AvformatOpenInput(&pFormatCtx, f.Edit1.Text(), ifmt, &options) < 0 {
			vcl.ShowMessage("Cannot open camera.\n")
			return
		}

		if pFormatCtx.AvformatFindStreamInfo(nil) < 0 {
			vcl.ShowMessage("Couldn't find stream information.")
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
			vcl.ShowMessage("Didn't find a video stream.\n")
			return
		}
		pCodecCtxPara := pFormatCtx.GetStream(uint32(videoindex)).Codecpar
		pCodec = libavcodec.AvcodecFindDecoder(pCodecCtxPara.CodecId)
		if pCodec == nil {
			vcl.ShowMessage("Codec not found.\n")
			return
		}

		pCodecCtx = pCodec.AvcodecAllocContext3()
		if pCodecCtx == nil {
			vcl.ShowMessage("Cannot alloc valid decode codec context.\n")
			return
		}
		defer pCodecCtx.AvcodecClose()

		if pCodecCtx.AvcodecParametersToContext(pCodecCtxPara) < 0 {
			vcl.ShowMessage("Cannot initialize parameters.\n")
			return
		}

		if pCodecCtx.AvcodecOpen2(pCodec, nil) < 0 {
			vcl.ShowMessage("Could not open codec.\n")
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
		var screen_w, screen_h ffcommon.FInt = 640, 360
		// var mode *sdl.SDL_DisplayMode = new(sdl.SDL_DisplayMode)
		// if sdl.SDL_GetCurrentDisplayMode(0, mode) != 0 {
		// 	fmt.Printf("SDL: could not get current display mode - exiting:%s\n", sdl.SDL_GetError())
		// 	return -1
		// }
		//Half of the Desktop's width and height.
		// screen_w = pCodecCtx.Width
		// screen_h = pCodecCtx.Height
		screen_w = f.Panel1.Width()
		screen_h = f.Panel1.Height()
		window := sdl.SDL_CreateWindowFrom(f.Panel3.Handle())
		if window == nil {
			vcl.ShowMessage(fmt.Sprintf("SDL: could not create window - exiting:%s\n", sdl.SDL_GetError()))
			return
		}
		defer window.SDL_DestroyWindow()
		renderer := window.SDL_CreateRenderer(-1, 0)
		if renderer == nil {
			vcl.ShowMessage(fmt.Sprintf("SDL: could not create renderer - exiting:%s\n", sdl.SDL_GetError()))
			return
		}
		defer renderer.SDL_DestroyRenderer()
		texture := renderer.SDL_CreateTexture(sdl.SDL_PIXELFORMAT_YV12,
			sdl.SDL_TEXTUREACCESS_STREAMING,
			pCodecCtx.Width,
			pCodecCtx.Height)
		defer texture.SDL_DestroyTexture()
		var rect sdl.SDL_Rect
		rect.X = 0
		rect.Y = 0
		rect.W = screen_w
		rect.H = screen_h
		var rect2 sdl.SDL_Rect
		rect2.X = 0
		rect2.Y = 0
		rect2.W = pCodecCtx.Width
		rect2.H = pCodecCtx.Height
		packet := &libavcodec.AVPacket{}
		var img_convert_ctx *libswscale.SwsContext
		var ret int32
		img_convert_ctx = libswscale.SwsGetContext(pCodecCtx.Width, pCodecCtx.Height, pCodecCtx.PixFmt, pCodecCtx.Width, pCodecCtx.Height, libavutil.AV_PIX_FMT_YUV420P, libswscale.SWS_BICUBIC, nil, nil, nil)
		defer img_convert_ctx.SwsFreeContext()
		for {
			if pFormatCtx.AvReadFrame(packet) >= 0 {
				if int32(packet.StreamIndex) == videoindex {
					if pCodecCtx.AvcodecSendPacket(packet) < 0 {
						packet.AvPacketUnref()
						vcl.ShowMessage(fmt.Sprintf("pCodecCtx.AvcodecSendPacket(packet) < 0\n"))
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

						if false {
							texture.SDL_UpdateYUVTexture(&rect2,
								pFrameYUV.Data[0], pFrameYUV.Linesize[0],
								pFrameYUV.Data[1], pFrameYUV.Linesize[1],
								pFrameYUV.Data[2], pFrameYUV.Linesize[2])

							renderer.SDL_RenderClear()
							renderer.SDL_RenderCopy(texture, nil, &rect)
							renderer.SDL_RenderPresent()
						} else {
							texture.SDL_UpdateYUVTexture(nil,
								pFrameYUV.Data[0], pFrameYUV.Linesize[0],
								pFrameYUV.Data[1], pFrameYUV.Linesize[1],
								pFrameYUV.Data[2], pFrameYUV.Linesize[2])

							renderer.SDL_RenderClear()
							renderer.SDL_RenderCopy(texture, nil, nil)
							renderer.SDL_RenderPresent()
						}

					}
				}

				packet.AvFreePacket()
			} else {
				vcl.ShowMessage(fmt.Sprintf("pFormatCtx.AvReadFrame(packet) < 0\n"))
				return
			}
		}
	}()
}

// rtmp停止
func (f *TForm1) OnButton7Click(sender vcl.IObject) {

}

// vlc播放
func (f *TForm1) OnButton8Click(sender vcl.IObject) {
	// f.Edit2.SetText("http://www.w3school.com.cn/i/movie.mp4")
	// f.Edit2.SetText("rtmp://liteavapp.qcloud.com/live/liteavdemoplayerstreamid")
	go func() {
		player := libvlc.NewVLCMediaPlayer()
		if player == nil {
			vcl.ShowMessage(fmt.Sprint("创建MediaPlayer失败:", libvlc.ErrMsg()))
			return
		}
		player.SethWnd(f.Panel4.Handle())
		player.LoadFromURL(f.Edit2.Text())
		player.Play()
	}()
}

// vlc停止
func (f *TForm1) OnButton9Click(sender vcl.IObject) {

}

// vdk rtmp播放
func (f *TForm1) OnButton10Click(sender vcl.IObject) {

	go func() {
		c, err := rtmp.Dial(f.Edit3.Text())
		if err != nil {
			vcl.ShowMessage(fmt.Sprint("连接rtmp失败:", err))
			return
		}
		defer c.Close()

		for {
			packetAV, err := c.ReadPacket()
			if err != nil {
				vcl.ShowMessage(fmt.Sprint("读取Packet失败:", err))
				return
			}
			codeData, err := c.Streams()
			if err != nil {
				vcl.ShowMessage(fmt.Sprint("读取Streams失败:", err))
				return
			}
			if codeData[packetAV.Idx].Type() == av.H264 {
				codec := codeData[packetAV.Idx].(h264parser.CodecData)
				_ = codec

			}
		}
	}()
}

// vdk rtmp停止
func (f *TForm1) OnButton11Click(sender vcl.IObject) {

}

func (f *TForm1) OnFormCreate(sender vcl.IObject) {
	f.Button3.SetVisible(false)
	f.OnButton3Click(nil)
}

// 库 播放
func (f *TForm1) OnButton12Click(sender vcl.IObject) {
	// f.Edit4.SetText("http://www.w3school.com.cn/i/movie.mp4")
	//f.Edit4.SetText("http://devimages.apple.com/iphone/samples/bipbop/gear1/prog_index.m3u8")
	// f.Edit1.SetText("rtmp://liteavapp.qcloud.com/live/liteavdemoplayerstreamid")
	// videohelper.Init()
	// return
	// go func() {
	a = videohelper.Play(f.Edit4.Text(), f.Panel6.Handle())
	// b = videohelper.Play(f.Edit4.Text(), f.Panel5.Handle())
	// if aaa {
	// 	a = videohelper.Play(f.Edit4.Text(), f.Panel6.Handle())
	// } else {
	// 	fmt.Println("OnButton12Click")
	// 	b = videohelper.Play(f.Edit4.Text(), f.Panel5.Handle())
	// }
	// }()
	// videohelper.Play(f.Edit4.Text(), f.Panel5.Handle())
	// videohelper.Play(f.Edit4.Text(), f.Panel4.Handle())
	// videohelper.Play(f.Edit4.Text(), f.Panel3.Handle())
	// videohelper.Play(f.Edit4.Text(), f.Panel2.Handle())
	// videohelper.Play(f.Edit4.Text(), f.Panel1.Handle())
}

var a *videohelper.Playinfo
var b *videohelper.Playinfo

// 库 停止
func (f *TForm1) OnButton13Click(sender vcl.IObject) {
	// go func() {
	// 	videohelper.Stop(a)
	// 	videohelper.Stop(b)
	// }()
	videohelper.Stop(a)
	// videohelper.Stop(b)
	// if aaa {
	// 	go videohelper.Stop(a)
	// 	time.Sleep(time.Second)
	// 	f.Panel6.SetVisible(false)
	// 	f.Panel6.SetVisible(true)
	// } else {
	// 	go videohelper.Stop(b)
	// 	time.Sleep(time.Second)
	// 	f.Panel5.SetVisible(false)
	// 	f.Panel5.SetVisible(true)
	// }
	// aaa = !aaa
	// vcl.ShowMessage(f.Panel6.Caption() + ":" + f.Panel5.Caption())
	// 整个操作完成后，显示到UI上时使用vcl.ThreadSync切换到主线程中执行。
	return

}

func (f *TForm1) OnPanel6Click(sender vcl.IObject) {
	vcl.ShowMessage(f.Panel6.Caption() + ":" + f.Panel5.Caption())

}

var bb bool

func (f *TForm1) OnButton14Click(sender vcl.IObject) {
	if bb {
		f.Panel6.SetVisible(true)
		f.Panel5.SetVisible(true)
	} else {
		f.Panel6.SetVisible(false)
		f.Panel5.SetVisible(false)
	}
	bb = !bb
}

func (f *TForm1) OnButton15Click(sender vcl.IObject) {
	vcl.ShowMessage(fmt.Sprint(f.Panel6.Visible()))
}

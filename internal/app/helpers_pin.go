package app

const mediaToolsVersion = "9.0.2"
const mediaToolsURL = "https://github.com/GyanD/codexffmpeg/releases/download/9.0.2/ffmpeg-9.0.2-essentials_build.zip"
const mediaToolsSHA256 = "60f467265b1e312373dbcd92200c2618a74850f98d3d078e94296bb3fa2047ba"
const mediaToolsBytes int64 = 114768076

type mediaToolPin struct {
	Member, SHA256 string
	Size           int64
}

var mediaToolFiles = map[string]mediaToolPin{
	"ffmpeg.exe":  {Member: "ffmpeg-9.0.2-essentials_build/bin/ffmpeg.exe", SHA256: "3256173f3f8bffd7df12227c68adf68025edb1832273a9530688a7bb1ed8edec", Size: 105423872},
	"ffprobe.exe": {Member: "ffmpeg-9.0.2-essentials_build/bin/ffprobe.exe", SHA256: "f0d36ecbbdd3bcfac3efa078c96c7271c2e68b3810595552ac3b7f17e9a65c52", Size: 105221120},
}

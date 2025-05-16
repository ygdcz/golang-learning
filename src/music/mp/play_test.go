package mp

import "testing"

func TestPlay(t *testing.T) {
	// 测试支持的音乐类型 mp3
	Play("test.mp3", "mp3")

	// 测试支持的音乐类型 wav
	Play("test.wav", "wav")

	// 测试不支持的音乐类型
	Play("test.unknown", "unknown")
}

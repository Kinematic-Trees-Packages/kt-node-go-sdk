package ktrobotics

import "testing"

func TestImageFrameAndVisionSampleRoundTrip(t *testing.T) {
	frame := ImageFrame{
		Source: "opencv-video-file", Width: 4, Height: 1, Channels: 3,
		FrameNumber: 0, CapturedUnixNS: 123,
		Data: []byte{0, 0, 0, 1, 0, 1, 2, 0, 2, 3, 0, 3},
	}
	payload, err := EncodeImageSample(frame)
	if err != nil { t.Fatal(err) }
	summary, err := DecodeImageSampleSummary(payload, 12)
	if err != nil { t.Fatal(err) }
	if summary.Source != "opencv-video-file" || summary.FrameNumber != 0 || summary.CapturedUnixNS != 123 { t.Fatalf("bad metadata: %+v", summary) }
	if len(summary.DataShape) != 3 || summary.DataShape[0] != 1 || summary.DataShape[1] != 4 || summary.DataShape[2] != 3 { t.Fatalf("bad shape: %+v", summary.DataShape) }
	want := []byte{0,0,0,1,0,1,2,0,2,3,0,3}
	for i := range want { if summary.DataPrefix[i] != want[i] { t.Fatalf("prefix[%d]=%d want %d", i, summary.DataPrefix[i], want[i]) } }
}

func TestEncodeImageSampleRejectsInvalidShape(t *testing.T) {
	_, err := EncodeImageSample(ImageFrame{Source:"source", Width:0, Height:1, Channels:3, Data:[]byte{}})
	if err == nil { t.Fatal("expected invalid dimensions") }
	_, err = EncodeImageSample(ImageFrame{Source:"source", Width:1, Height:1, Channels:3, Data:[]byte{1,2}})
	if err == nil { t.Fatal("expected invalid data length") }
}

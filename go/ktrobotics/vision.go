package ktrobotics

import (
	"fmt"

	data "github.com/kinematic-trees/librobotics-language-video-nodes/sdk/go/ktrobotics/generated/bow/data"
	flatbuffers "github.com/google/flatbuffers/go"
)

type ImageFrame struct {
	Source         string
	Width          int
	Height         int
	Channels       int
	FrameNumber    uint64
	CapturedUnixNS uint64
	Data           []byte
}

type ImageSummary struct {
	Source         string
	FrameNumber    uint64
	DataShape      []uint32
	DataPrefix     []byte
	CapturedUnixNS uint64
	Compression    string
	ImageType      string
	Pipeline       string
}

func EncodeImageSample(frame ImageFrame) ([]byte, error) {
	if frame.Width <= 0 || frame.Height <= 0 || frame.Channels <= 0 { return nil, fmt.Errorf("invalid frame shape") }
	if len(frame.Data) != frame.Width*frame.Height*frame.Channels { return nil, fmt.Errorf("frame data length %d does not match shape", len(frame.Data)) }
	builder := flatbuffers.NewBuilder(len(frame.Data) + 256)
	source := builder.CreateString(frame.Source)
	dataVec := builder.CreateByteVector(frame.Data)
	shape := []uint32{uint32(frame.Height), uint32(frame.Width), uint32(frame.Channels)}
	data.ImageSampleStartDataShapeVector(builder, len(shape))
	for i := len(shape) - 1; i >= 0; i-- { builder.PrependUint32(shape[i]) }
	shapeVec := builder.EndVector(len(shape))
	data.ImageSampleStart(builder)
	data.ImageSampleAddSource(builder, source)
	data.ImageSampleAddData(builder, dataVec)
	data.ImageSampleAddDataShape(builder, shapeVec)
	data.ImageSampleAddCompression(builder, data.CompressionFormatRAW)
	data.ImageSampleAddImageType(builder, data.ImageTypeRGB)
	data.ImageSampleAddFrameNumber(builder, frame.FrameNumber)
	data.ImageSampleAddDesignation(builder, data.StereoDesignationNONE)
	data.ImageSampleAddPipeline(builder, data.MediaPipelineOTHER)
	data.ImageSampleAddCapturedUnixNs(builder, frame.CapturedUnixNS)
	data.ImageSampleAddDepthRepresentation(builder, data.DepthRepresentationUNSPECIFIED)
	data.ImageSampleAddDepthColorization(builder, data.DepthColorizationNONE)
	data.ImageSampleAddNewDataFlag(builder, true)
	sample := data.ImageSampleEnd(builder)
	builder.FinishWithFileIdentifier(sample, []byte("VSM1"))
	return append([]byte(nil), builder.FinishedBytes()...), nil
}

func DecodeImageSampleSummary(payload []byte, prefix int) (ImageSummary, error) {
	if len(payload) < 8 { return ImageSummary{}, fmt.Errorf("payload too small") }
	sample := data.GetRootAsImageSample(payload, 0)
	shape := make([]uint32, sample.DataShapeLength())
	for i := range shape { shape[i] = sample.DataShape(i) }
	if prefix < 0 { prefix = 0 }
	if prefix > sample.DataLength() { prefix = sample.DataLength() }
	bytes := make([]byte, prefix)
	copy(bytes, sample.DataBytes()[:prefix])
	return ImageSummary{
		Source: string(sample.Source()), FrameNumber: sample.FrameNumber(), DataShape: shape,
		DataPrefix: bytes, CapturedUnixNS: sample.CapturedUnixNs(),
		Compression: sample.Compression().String(), ImageType: sample.ImageType().String(), Pipeline: sample.Pipeline().String(),
	}, nil
}

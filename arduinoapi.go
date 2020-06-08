package nango

const (
	PinLow = iota
	PinHigh
)

const (
	PinInput = iota
	PinOutput
	PinInputPullup
)

const (
	LsbFirst = iota
	MsbFirst
)

type ArduinoApi struct {
	*FirmwareClass
}

func NewArduinoApi(conn *FirmwareConnection) *ArduinoApi {
	return &ArduinoApi{
		&FirmwareClass{
			Conn:      conn,
			Id:        0,
			Namespace: "A",
		}}
}

func (api *ArduinoApi) DigitalWrite(pin string, val int) error {
	return api.CallAndReturnNothing("dw", pin, val)
}

func (api *ArduinoApi) DigitalRead(pin string) (int, error) {
	return api.CallAndReturnInt("r", pin)
}

func (api *ArduinoApi) AnalogWrite(pin string, val int) error {
	return api.CallAndReturnNothing("aw", pin)
}

func (api *ArduinoApi) AnalogRead(pin string) (int, error) {
	return api.CallAndReturnInt("a", pin)
}

func (api *ArduinoApi) PinMode(pin string, mode int) error {
	return api.CallAndReturnNothing("pm", pin, mode)
}

func (api *ArduinoApi) Millis() (int, error) {
	return api.CallAndReturnInt("m")
}

func (api *ArduinoApi) PulseIn(pin string, val int) (int, error) {
	return api.CallAndReturnInt("pi", pin, val)
}

func (api *ArduinoApi) ShiftOut(dataPin string, clockPin string, bitOrder int, val byte) (int, error) {
	return api.CallAndReturnInt("s", dataPin, clockPin, bitOrder, val)
}

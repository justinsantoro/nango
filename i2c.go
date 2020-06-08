package nango

import "log"

type I2CAddress int

func (addr *I2CAddress) Value() interface{} {
	if addr != nil {
		return int(*addr)
	}
	return nil
}

type i2cCommunicationError int

func (d i2cCommunicationError) String() string {
	return [...]string{
		"data too long to fit in transmit buffer",
		"received NACK on transmit of address",
		"received NACK on transmit of data",
		"other error",
	}[d-1]
}

func (d i2cCommunicationError) Error() string {
	return d.String()
}

type i2cbase struct {
	wire           *wire
	Address        *I2CAddress
	busInitialized bool
}

func newI2cBase(wire *wire, address *I2CAddress) *i2cbase {
	return &i2cbase{
		wire:           wire,
		Address:        address,
		busInitialized: false,
	}
}

func (b *i2cbase) begin() error {
	if !b.busInitialized {
		return b.wire.Begin(b.Address)
	}
	return nil
}

type I2CMaster struct {
	*i2cbase
}

func NewI2cMaster(wire *wire) *I2CMaster {
	return &I2CMaster{
		newI2cBase(wire, nil),
	}
}

func (m *I2CMaster) Request(address I2CAddress, quantity int) ([]byte, error) {
	err := m.begin()
	if err != nil {
		return nil, err
	}
	n, err := m.wire.RequestFrom(address, quantity, true)
	if n < quantity {
		log.Println("i2cMaster: slave sent less bytes than requested")
	}
	buf := make([]byte, n)
	_, err = m.wire.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (m *I2CMaster) Send(address I2CAddress, data []byte) error {
	err := m.begin()
	if err != nil {
		return err
	}
	err = m.wire.BeginTransmission(address)
	if err != nil {
		return nil
	}
	_, err = m.wire.Write(data)
	if err != nil {
		return err
	}
	c, err := m.wire.EndTransmission(true)
	if err != nil {
		return err
	}
	if c != 0 {
		return i2cCommunicationError(c)
	}
	return nil
}

func (m *I2CMaster) Scan() ([]I2CAddress, error) {
	err := m.begin()
	if err != nil {
		return nil, err
	}
	addrs := make([]I2CAddress, 0)
	for i := 1; i <= 128; i++ {
		addr := I2CAddress(i)
		err = m.Send(addr, make([]byte, 0))
		if err != nil {
			switch err.(type) {
			case i2cCommunicationError:
				//no response
				continue
			default:
				//firmware communication error
				return nil, err
			}
		}
		addrs = append(addrs, addr)
	}
	return addrs, nil
}

type I2CSlave struct {
	*i2cbase
}

func NewI2cSlave(wire *wire, addr I2CAddress) *I2CSlave {
	return &I2CSlave{newI2cBase(wire, &addr)}
}

func (s *I2CSlave) Receive() (err error) {
	err = s.begin()
	if err != nil {
		return
	}
	var n int
	n, err = s.wire.Available()
	if err != nil {
		return
	}
	buf := make([]byte, n)
	_, err = s.wire.Read(buf)
	return
}

func (s *I2CSlave) Write(b []byte) (i int, err error) {
	err = s.begin()
	if err != nil {
		return
	}
	return s.wire.Write(b)
}

type wire struct {
	*FirmwareClass
}

//NewWire returns a wire struct giving access to the arduino wire library
//http://arduino.cc/en/reference/wire
func NewWire(conn *FirmwareConnection) *wire {
	return &wire{
		&FirmwareClass{
			Conn:      conn,
			Id:        0,
			Namespace: "Wire",
		},
	}
}

//Begin initiates the Wire library and joins the I2C bus as a master or slave.
//This should normally only be called once.
//Pass nil to join the bus as a master
func (w *wire) Begin(address *I2CAddress) error {
	return w.CallAndReturnNothing("begin", address.Value())
}

func (w *wire) RequestFrom(address I2CAddress, quantity int, stop bool) (int, error) {
	return w.CallAndReturnInt("requestFrom", address.Value(), quantity, stop)
}

func (w *wire) BeginTransmission(address I2CAddress) error {
	_, err := w.call("beginTransmission", address.Value())
	return err
}

func (w *wire) EndTransmission(stop bool) (int, error) {
	return w.CallAndReturnInt("endTransmission", stop)
}

func (w *wire) Write(b []byte) (i int, err error) {
	var v byte
	for i, v = range b {
		err = w.CallAndReturnNothing("write", v)
		if err != nil {
			return
		}
	}
	return
}

func (w *wire) Available() (int, error) {
	return w.CallAndReturnInt("available")
}

func (w *wire) Read(b []byte) (i int, err error) {
	var v byte
	for i = 0; i < len(b); i++ {
		v, err = w.CallAndReturnByte("read")
		if err != nil {
			return
		}
		b[i] = v
	}
	return
}

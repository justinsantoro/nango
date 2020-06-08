package nango

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/justinsantoro/nango/serial"
	"log"
	"strconv"
	"sync"
	"time"
)

type SerialTimeoutError string

func (t SerialTimeoutError) String() string {
	return string(t)
}

func (t SerialTimeoutError) Error() string {
	return t.String()
}

var mutex = new(sync.Mutex)

type FirmwareConnection struct {
	readWriter        *bufio.ReadWriter
	SerialConfig      *serial.Config
	SleepAfterConnect time.Duration
	ReadTimeout       time.Duration
	port              *serial.Port
}

func NewFirmwareConnection(serialConf *serial.Config) *FirmwareConnection {
	return &FirmwareConnection{
		SerialConfig:      serialConf,
		SleepAfterConnect: 0,
		port:              nil,
		ReadTimeout:       2 * time.Second,
	}
}

func (s *FirmwareConnection) Open() error {
	//log.Printf("opening port:%v [%v baud]\n", s.SerialConfig.Name, s.SerialConfig.Baud)
	var err error
	s.port, err = serial.OpenPort(s.SerialConfig)
	if err != nil {
		return err
	}
	s.readWriter = bufio.NewReadWriter(bufio.NewReader(s.port), bufio.NewWriter(s.port))
	//log.Println("port opened successfully")
	time.Sleep(s.SleepAfterConnect)
	return s.port.Flush()
}

func (s *FirmwareConnection) Write(b []byte) error {
	if s.port == nil {
		return portClosed()
	}
	_, err := s.readWriter.Write(b)
	if err != nil {
		return err
	}
	//log.Printf("successfully wrote %v bytes to port %v\n", i, s.SerialConfig.Name)
	return nil
}

func (s *FirmwareConnection) Flush() error {
	if s.port == nil {
		return portClosed()
	}
	return s.readWriter.Flush()
}

func (s *FirmwareConnection) ReadLine() (b []byte, err error) {
	if s.port == nil {
		err = portClosed()
		return
	}
	scanner := bufio.NewScanner(s.readWriter.Reader)
	b = make([]byte, 0)
	errChan := make(chan error)
	go func() {
		scanner.Scan()
		errChan <- scanner.Err()
	}()
	select {
	case err = <-errChan:
		if err != nil {
			err = errors.New(fmt.Sprintf("error scanning bytes from port %s\n: %s", s.SerialConfig.Name, err))
		}
	case <-time.After(s.ReadTimeout):
		err = SerialTimeoutError(s.SerialConfig.Name + " ReadLine timeout")
	}
	//if there was an error, flush the port
	if err != nil {
		errFlush := s.port.Flush()
		if errFlush != nil {
			log.Printf("error flushing serial port %s: %s", s.SerialConfig.Name, errFlush)
		}
		return
	}
	//log.Printf("successfully read line of %v bytes from port %v\n", len(b), s.SerialConfig.Name)
	b = scanner.Bytes()
	return
}

func (s *FirmwareConnection) FlushPort() error {
	if s.port == nil {
		return portClosed()
	}
	return s.port.Flush()
}

func (s *FirmwareConnection) Close() error {
	if s.port == nil {
		return nil
	}
	return s.port.Close()
}

func portClosed() error {
	return errors.New("port is not opened: must call Open() first")
}

func write(data interface{}, conn *FirmwareConnection) error {
	var s string
	switch v := data.(type) {
	case string:
		s = v
	case int:
		s = strconv.Itoa(v)
	case bool:
		//encode bool types as Python string representations of booleans
		switch v {
		case true:
			s = "True"
		default:
			s = "False"
		}
	default:
		return errors.New(fmt.Sprintf("Firmware Write: Unsupported type %T", v))
	}
	//fmt.Printf(s)
	return conn.Write([]byte(s + "\000"))
}

func returnValue(conn *FirmwareConnection) (v string, err error) {
	b, err := conn.ReadLine()
	if err != nil {
		return
	}
	v = string(b)
	return
}

func call(namespace string, id int, args []interface{}, conn *FirmwareConnection) (v string, err error) {
	toprint := []interface{}{}
	nel := 0

	mutex.Lock()
	defer mutex.Unlock()

	err = write(namespace, conn)
	if err != nil {
		return
	}
	err = write(id, conn)
	if err != nil {
		return
	}

	for _, arg := range args {
		if ls, ok := arg.([]interface{}); ok {
			for _, el := range ls {
				if el != nil {
					toprint = append(toprint, el)
					nel++
				}
			}
		} else {
			if arg != nil {
				toprint = append(toprint, arg)
				nel++
			}
		}
	}

	err = write(nel-1, conn)
	if err != nil {
		return
	}

	for _, elprint := range toprint {
		err = write(elprint, conn)
		if err != nil {
			return
		}
	}
	err = conn.Flush()
	if err != nil {
		return
	}
	return returnValue(conn)
}

func prependName(args []interface{}, name string) []interface{} {
	args = append(args, 0)
	copy(args[1:], args)
	args[0] = name
	return args
}

func ArduinoMethodCall(f *FirmwareClass, methodName string, args ...interface{}) (string, error) {
	return call(f.Namespace, f.Id, prependName(args, methodName), f.conn())
}

type FirmwareClass struct {
	Conn      *FirmwareConnection
	Id        int
	Namespace string
}

func (f *FirmwareClass) conn() *FirmwareConnection {
	return f.Conn
}

func (f *FirmwareClass) call(methodName string, args ...interface{}) (string, error) {
	return ArduinoMethodCall(f, methodName, args)
}

func (f *FirmwareClass) CallAndReturnByte(methodName string, args ...interface{}) (byte, error) {
	s, err := ArduinoMethodCall(f, methodName, args)
	if len(s) > 1 {
		log.Println("warning: callAndReturnByte received more than 1 byte")
	}
	return s[0], err
}

func (f *FirmwareClass) CallAndReturnInt(methodName string, args ...interface{}) (int, error) {
	s, err := ArduinoMethodCall(f, methodName, args)
	if err != nil {
		return -1, err
	}
	return strconv.Atoi(s)
}

func (f *FirmwareClass) CallAndReturnFloat(methodName string, args ...interface{}) (float64, error) {
	s, err := ArduinoMethodCall(f, methodName, args)
	if err != nil {
		return -1, err
	}
	return strconv.ParseFloat(s, 64)
}

func (f *FirmwareClass) CallAndReturnNothing(methodName string, args ...interface{}) error {
	_, err := ArduinoMethodCall(f, methodName, args)
	return err
}

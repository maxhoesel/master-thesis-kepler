// Copyright 2018 The go-libvirt Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//
// Code generated by internal/lvgen/generate.go. DO NOT EDIT.
//
// To regenerate, run 'go generate' in internal/lvgen.
//

package libvirt

import (
	"bytes"
	"io"

	"github.com/digitalocean/go-libvirt/internal/constants"
	"github.com/digitalocean/go-libvirt/internal/go-xdr/xdr2"
)

// References to prevent "imported and not used" errors.
var (
	_ = bytes.Buffer{}
	_ = io.Copy
	_ = constants.Program
	_ = xdr.Unmarshal
)

//
// Typedefs:
//

//
// Enums:
//
// QEMUProcedure is libvirt's qemu_procedure
type QEMUProcedure int32

//
// Structs:
//
// QEMUDomainMonitorCommandArgs is libvirt's qemu_domain_monitor_command_args
type QEMUDomainMonitorCommandArgs struct {
	Dom Domain
	Cmd string
	Flags uint32
}

// QEMUDomainMonitorCommandRet is libvirt's qemu_domain_monitor_command_ret
type QEMUDomainMonitorCommandRet struct {
	Result string
}

// QEMUDomainAttachArgs is libvirt's qemu_domain_attach_args
type QEMUDomainAttachArgs struct {
	PidValue uint32
	Flags uint32
}

// QEMUDomainAttachRet is libvirt's qemu_domain_attach_ret
type QEMUDomainAttachRet struct {
	Dom Domain
}

// QEMUDomainAgentCommandArgs is libvirt's qemu_domain_agent_command_args
type QEMUDomainAgentCommandArgs struct {
	Dom Domain
	Cmd string
	Timeout int32
	Flags uint32
}

// QEMUDomainAgentCommandRet is libvirt's qemu_domain_agent_command_ret
type QEMUDomainAgentCommandRet struct {
	Result OptString
}

// QEMUConnectDomainMonitorEventRegisterArgs is libvirt's qemu_connect_domain_monitor_event_register_args
type QEMUConnectDomainMonitorEventRegisterArgs struct {
	Dom OptDomain
	Event OptString
	Flags uint32
}

// QEMUConnectDomainMonitorEventRegisterRet is libvirt's qemu_connect_domain_monitor_event_register_ret
type QEMUConnectDomainMonitorEventRegisterRet struct {
	CallbackID int32
}

// QEMUConnectDomainMonitorEventDeregisterArgs is libvirt's qemu_connect_domain_monitor_event_deregister_args
type QEMUConnectDomainMonitorEventDeregisterArgs struct {
	CallbackID int32
}

// QEMUDomainMonitorEventMsg is libvirt's qemu_domain_monitor_event_msg
type QEMUDomainMonitorEventMsg struct {
	CallbackID int32
	Dom Domain
	Event string
	Seconds int64
	Micros uint32
	Details OptString
}




// QEMUDomainMonitorCommand is the go wrapper for QEMU_PROC_DOMAIN_MONITOR_COMMAND.
func (l *Libvirt) QEMUDomainMonitorCommand(Dom Domain, Cmd string, Flags uint32) (rResult string, err error) {
	var buf []byte

	args := QEMUDomainMonitorCommandArgs {
		Dom: Dom,
		Cmd: Cmd,
		Flags: Flags,
	}

	buf, err = encode(&args)
	if err != nil {
		return
	}

	var r response
	r, err = l.requestStream(1, constants.QEMUProgram, buf, nil, nil)
	if err != nil {
		return
	}

	// Return value unmarshaling
	tpd := typedParamDecoder{}
	ct := map[string]xdr.TypeDecoder{"libvirt.TypedParam": tpd}
	rdr := bytes.NewReader(r.Payload)
	dec := xdr.NewDecoderCustomTypes(rdr, 0, ct)
	// Result: string
	_, err = dec.Decode(&rResult)
	if err != nil {
		return
	}

	return
}

// QEMUDomainAttach is the go wrapper for QEMU_PROC_DOMAIN_ATTACH.
func (l *Libvirt) QEMUDomainAttach(PidValue uint32, Flags uint32) (rDom Domain, err error) {
	var buf []byte

	args := QEMUDomainAttachArgs {
		PidValue: PidValue,
		Flags: Flags,
	}

	buf, err = encode(&args)
	if err != nil {
		return
	}

	var r response
	r, err = l.requestStream(2, constants.QEMUProgram, buf, nil, nil)
	if err != nil {
		return
	}

	// Return value unmarshaling
	tpd := typedParamDecoder{}
	ct := map[string]xdr.TypeDecoder{"libvirt.TypedParam": tpd}
	rdr := bytes.NewReader(r.Payload)
	dec := xdr.NewDecoderCustomTypes(rdr, 0, ct)
	// Dom: Domain
	_, err = dec.Decode(&rDom)
	if err != nil {
		return
	}

	return
}

// QEMUDomainAgentCommand is the go wrapper for QEMU_PROC_DOMAIN_AGENT_COMMAND.
func (l *Libvirt) QEMUDomainAgentCommand(Dom Domain, Cmd string, Timeout int32, Flags uint32) (rResult OptString, err error) {
	var buf []byte

	args := QEMUDomainAgentCommandArgs {
		Dom: Dom,
		Cmd: Cmd,
		Timeout: Timeout,
		Flags: Flags,
	}

	buf, err = encode(&args)
	if err != nil {
		return
	}

	var r response
	r, err = l.requestStream(3, constants.QEMUProgram, buf, nil, nil)
	if err != nil {
		return
	}

	// Return value unmarshaling
	tpd := typedParamDecoder{}
	ct := map[string]xdr.TypeDecoder{"libvirt.TypedParam": tpd}
	rdr := bytes.NewReader(r.Payload)
	dec := xdr.NewDecoderCustomTypes(rdr, 0, ct)
	// Result: OptString
	_, err = dec.Decode(&rResult)
	if err != nil {
		return
	}

	return
}

// QEMUConnectDomainMonitorEventRegister is the go wrapper for QEMU_PROC_CONNECT_DOMAIN_MONITOR_EVENT_REGISTER.
func (l *Libvirt) QEMUConnectDomainMonitorEventRegister(Dom OptDomain, Event OptString, Flags uint32) (rCallbackID int32, err error) {
	var buf []byte

	args := QEMUConnectDomainMonitorEventRegisterArgs {
		Dom: Dom,
		Event: Event,
		Flags: Flags,
	}

	buf, err = encode(&args)
	if err != nil {
		return
	}

	var r response
	r, err = l.requestStream(4, constants.QEMUProgram, buf, nil, nil)
	if err != nil {
		return
	}

	// Return value unmarshaling
	tpd := typedParamDecoder{}
	ct := map[string]xdr.TypeDecoder{"libvirt.TypedParam": tpd}
	rdr := bytes.NewReader(r.Payload)
	dec := xdr.NewDecoderCustomTypes(rdr, 0, ct)
	// CallbackID: int32
	_, err = dec.Decode(&rCallbackID)
	if err != nil {
		return
	}

	return
}

// QEMUConnectDomainMonitorEventDeregister is the go wrapper for QEMU_PROC_CONNECT_DOMAIN_MONITOR_EVENT_DEREGISTER.
func (l *Libvirt) QEMUConnectDomainMonitorEventDeregister(CallbackID int32) (err error) {
	var buf []byte

	args := QEMUConnectDomainMonitorEventDeregisterArgs {
		CallbackID: CallbackID,
	}

	buf, err = encode(&args)
	if err != nil {
		return
	}


	_, err = l.requestStream(5, constants.QEMUProgram, buf, nil, nil)
	if err != nil {
		return
	}

	return
}

// QEMUDomainMonitorEvent is the go wrapper for QEMU_PROC_DOMAIN_MONITOR_EVENT.
func (l *Libvirt) QEMUDomainMonitorEvent() (err error) {
	var buf []byte


	_, err = l.requestStream(6, constants.QEMUProgram, buf, nil, nil)
	if err != nil {
		return
	}

	return
}


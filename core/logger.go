//     Copyright (C) 2020-2021, IrineSistiana
//
//     This file is part of simple-tls.
//
//     simple-tls is free software: you can redistribute it and/or modify
//     it under the terms of the GNU General Public License as published by
//     the Free Software Foundation, either version 3 of the License, or
//     (at your option) any later version.
//
//     simple-tls is distributed in the hope that it will be useful,
//     but WITHOUT ANY WARRANTY; without even the implied warranty of
//     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//     GNU General Public License for more details.
//
//     You should have received a copy of the GNU General Public License
//     along with this program.  If not, see <https://www.gnu.org/licenses/>.

package core

import (
	"log"
	"net"
	"net/http"
	"os"
)

var errLogger = log.New(os.Stderr, "err", log.LstdFlags)

func logConnErr(conn net.Conn, err error) {
	errLogger.Printf("connection %s <-> %s: %v", conn.RemoteAddr(), conn.LocalAddr(), err)
}

func logRequestErr(r *http.Request, err error) {
	errLogger.Printf("request from %s %s: %v", r.RemoteAddr, r.RequestURI, err)
}
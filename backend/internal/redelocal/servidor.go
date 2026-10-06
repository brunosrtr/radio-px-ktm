// Package redelocal anuncia o backend na rede e fornece pareamento offline.
package redelocal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
	qrcode "github.com/skip2/go-qrcode"
)

const TipoServico = "_radiopx._tcp"

type Servidor struct {
	ID    string
	Nome  string
	Porta int
}

func Identidade(caminho string) (string, error) {
	b, err := os.ReadFile(caminho)
	if err == nil {
		id := strings.TrimSpace(string(b))
		if len(id) != 32 {
			return "", fmt.Errorf("identidade local inválida em %s", caminho)
		}
		if _, err := hex.DecodeString(id); err != nil {
			return "", err
		}
		return id, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	b = make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	id := hex.EncodeToString(b)
	f, err := os.OpenFile(caminho, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		return Identidade(caminho)
	}
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(id); err != nil {
		f.Close()
		return "", err
	}
	return id, f.Close()
}

// Interfaces físicas ou VPNs podem mudar enquanto o backend está em execução.
// Anunciamos apenas IPv4 de interfaces ativas, sem loopback ou bridges Docker.
func interfacesLocais() ([]net.Interface, []string) {
	interfaces, _ := net.Interfaces()
	var selecionadas []net.Interface
	var ips []string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		if strings.HasPrefix(iface.Name, "docker") || strings.HasPrefix(iface.Name, "veth") || strings.HasPrefix(iface.Name, "br-") {
			continue
		}
		enderecos, err := iface.Addrs()
		if err != nil {
			continue
		}
		temIP := false
		for _, endereco := range enderecos {
			ip, _, err := net.ParseCIDR(endereco.String())
			if err != nil || ip.To4() == nil || !ip.IsGlobalUnicast() {
				continue
			}
			ips = append(ips, ip.String())
			temIP = true
		}
		if temIP {
			selecionadas = append(selecionadas, iface)
		}
	}
	sort.Strings(ips)
	return selecionadas, ips
}

func (s *Servidor) Enderecos() []string {
	_, ips := interfacesLocais()
	enderecos := make([]string, 0, len(ips))
	for _, ip := range ips {
		enderecos = append(enderecos, "http://"+net.JoinHostPort(ip, strconv.Itoa(s.Porta)))
	}
	return enderecos
}

func (s *Servidor) Anunciar(ctx context.Context) {
	var anuncio *zeroconf.Server
	var assinatura string
	defer func() {
		if anuncio != nil {
			anuncio.Shutdown()
		}
	}()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		interfaces, ips := interfacesLocais()
		nova := strings.Join(ips, ",")
		for _, iface := range interfaces {
			nova += fmt.Sprintf("/%d", iface.Index)
		}
		if anuncio == nil || nova != assinatura {
			if anuncio != nil {
				anuncio.Shutdown()
				anuncio = nil
			}
			if len(ips) > 0 {
				var err error
				anuncio, err = zeroconf.RegisterProxy(s.Nome+"-"+s.ID[:8], TipoServico, "local.",
					s.Porta, "radiopx-"+s.ID[:8], ips, []string{"id=" + s.ID, "version=1"}, interfaces)
				if err != nil {
					log.Printf("descoberta local indisponível (QR Code continua disponível): %v", err)
				}
			}
			assinatura = nova
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Servidor) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(map[string]any{
			"service": "radio-px-ktm", "version": 1, "id": s.ID, "name": s.Nome,
			"addresses": s.Enderecos(),
		})
	})
	mux.HandleFunc("GET /qr", func(w http.ResponseWriter, r *http.Request) {
		endereco := r.URL.Query().Get("endereco")
		valido := false
		for _, atual := range s.Enderecos() {
			if endereco == atual {
				valido = true
				break
			}
		}
		if !valido {
			http.Error(w, "endereço não pertence à rede atual", http.StatusBadRequest)
			return
		}
		dados, _ := json.Marshal(map[string]any{"service": "radio-px-ktm", "version": 1, "id": s.ID, "url": endereco})
		png, err := qrcode.Encode(string(dados), qrcode.Medium, 320)
		if err != nil {
			http.Error(w, "não foi possível gerar QR Code", 500)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(png)
	})
	return mux
}

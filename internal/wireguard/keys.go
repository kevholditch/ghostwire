package wireguard

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func GeneratePrivateKey(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "wg", "genkey").Output()
	if err != nil {
		return "", fmt.Errorf("wg genkey: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func PublicKey(ctx context.Context, privateKey string) (string, error) {
	cmd := exec.CommandContext(ctx, "wg", "pubkey")
	cmd.Stdin = strings.NewReader(privateKey + "\n")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("wg pubkey: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

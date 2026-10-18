#!/usr/bin/env bash
# Baseline gate Fase B: dipakai sebelum dan sesudah perubahan.
set -u
cd /e/rekapProject/projecct_build_with_AI/server || exit 1

echo "--- gofmt -l . ---"
gofmt -l . > /tmp/gofmt_out.txt 2>&1
if [ -s /tmp/gofmt_out.txt ]; then
  echo "TIDAK BERSIH:"; cat /tmp/gofmt_out.txt
else
  echo "(kosong)"
fi

echo "--- go vet ./... ---"
go vet ./... > /tmp/vet_out.txt 2>&1
echo "exit=$?"
cat /tmp/vet_out.txt

echo "--- go test -count=1 ./... ---"
go test -count=1 ./... > /tmp/test_out.txt 2>&1
echo "exit=$?"
grep -c '^ok' /tmp/test_out.txt | sed 's/^/paket ok: /'
grep -c '^FAIL' /tmp/test_out.txt | sed 's/^/paket FAIL: /'
cat /tmp/test_out.txt

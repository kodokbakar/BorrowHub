# Go Testing

## Unit test

- File menggunakan suffix `_test.go`.
- Nama test menjelaskan perilaku, bukan implementasi.
- Gunakan standard library `testing`.
- Gunakan table test hanya ketika beberapa kasus menjalankan alur yang sama.
- Hindari mock jika fungsi murni atau dependency nyata lebih sederhana.
- Gunakan `t.Helper()` untuk helper test.
- Unit test tidak boleh membutuhkan network, database, atau environment lokal.

Jalankan:

```bash
make test
```

## Integration test

Integration test menggunakan build tag:

```go
//go:build integration
```

Integration test boleh memakai PostgreSQL dan dependency eksternal lokal.

Jalankan:

```bash
make test-integration
```
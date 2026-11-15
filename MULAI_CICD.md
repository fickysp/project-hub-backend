# Mulai CI/CD ProjectHub ke VM

Paket ini memperbaiki deployment pada source `main` commit `2db7943` (15 November 2026). Isinya adalah file pengganti, bukan seluruh repository. CORS tambahan memakai FRONTEND_URL sebagai satu origin yang diizinkan. Cache yang sudah ada di main dipertahankan.

Dockerfile lama menyalin folder migrations yang tidak ada; Compose lama merujuk service postgres yang tidak didefinisikan dan menyimpan password database langsung di source. Paket ini memperbaiki ketiganya. Ganti password yang pernah tercantum di source melalui Supabase dan perbarui DATABASE_URL di semua deployment yang memakainya. Menghapus password dari commit terbaru tidak menghapusnya dari riwayat Git.

## 1. Pilih kelompok dan SSH dari Mac

| Kelompok | Port SSH | Frontend | Backend |
|---|---|---|---|
| 1 | 11111 | https://group1.bccdev.id | https://group1-be.bccdev.id |
| 2 | 11112 | https://group2.bccdev.id | https://group2-be.bccdev.id |
| 3 | 11113 | https://group3.bccdev.id | https://group3-be.bccdev.id |
| 4 | 11114 | https://group4.bccdev.id | https://group4-be.bccdev.id |
| 5 | 11115 | https://group5.bccdev.id | https://group5-be.bccdev.id |
| 6 | 11116 | https://group6.bccdev.id | https://group6-be.bccdev.id |

Contoh berikut memakai kelompok 2; sesuaikan file key dan port dengan kelompokmu. Jalankan dari folder tempat private key panitia disimpan:

```bash
chmod 600 ./group2
ssh -i ./group2 -p 11112 dev@proxy.bccdev.id
```

Private key adalah file `group2`, bukan `group2.pub`. Jangan commit private key ke repository.

## 2. Cek VM, jalankan di terminal SSH

```bash
cat /etc/os-release
uname -m
docker --version
docker compose version
ss -ltnp | grep ':8080 ' || true
```

Backend memakai port VM 8080; frontend memakai 80. Jangan menjalankan dua backend pada port 8080. Jika sudah dipakai, identifikasi layanannya sebelum memulai container baru. Panduan instalasi berikut khusus Ubuntu; untuk OS lain sesuaikan instalasi Docker terlebih dahulu.

`x86_64` berarti platform image `linux/amd64` (default workflow). Jika output `aarch64`, tambahkan repository variable `VM_PLATFORM=linux/arm64` di GitHub Settings → Secrets and variables → Actions → Variables.

## 3. Install Docker jika belum tersedia — Ubuntu

Jika Docker dan Compose sudah berfungsi, lewati instalasi ini.

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc
. /etc/os-release
printf 'deb [arch=%s signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu %s stable\n' "$(dpkg --print-architecture)" "$VERSION_CODENAME" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo systemctl enable --now docker
sudo usermod -aG docker dev
exit
```

Login SSH lagi dari Mac agar grup docker diterapkan:

```bash
ssh -i ./group2 -p 11112 dev@proxy.bccdev.id
```

Di dalam VM:

```bash
docker info >/dev/null
docker compose version
```

Workflow memakai Docker tanpa sudo. User dalam grup Docker memiliki hak yang setara root terhadap host. Jangan menambahkan user lain yang tidak dibutuhkan.

## 4. Siapkan .env hanya di VM

```bash
mkdir -p "$HOME/projecthub-backend/incoming"
chmod 700 "$HOME/projecthub-backend"
umask 077
nano "$HOME/projecthub-backend/.env"
```

Isi mengikuti VM.env.example di paket. Untuk kelompok 2, FRONTEND_URL harus `https://group2.bccdev.id`. DATABASE_URL harus variabel persis itu; `DB_URL` tidak dibaca source ini. Gunakan connection string Supabase milikmu yang berlaku dengan TLS sesuai instruksi Supabase; jangan menyalin kredensial contoh lama dari source. Startup membutuhkan JWT_SECRET, REFRESH_SECRET, SUPABASE_URL dan SUPABASE_KEY. Buat dua secret JWT berbeda dengan `openssl rand -hex 32`.

Simpan di nano dengan Ctrl+O → Enter → Ctrl+X, lalu:

```bash
chmod 600 "$HOME/projecthub-backend/.env"
```

Jangan kirim isi .env/private key ke chat dan jangan commit ke GitHub. Database aplikasi tetap di Supabase; paket ini tidak membuat PostgreSQL baru di VM dan tidak menghapus data. AutoMigrate masih berjalan saat aplikasi mulai. Uji schema data lama/backup sebelum deploy; pipeline ini tidak menambahkan konversi skills lama atau perbaikan schema aplikasi.

## 5. Rekam SSH host key dari Mac

Keluar dari SSH dengan `exit`. Di Mac:

```bash
ssh-keyscan -p 11112 proxy.bccdev.id > group2-known-hosts
ssh-keygen -lf group2-known-hosts
```

Cocokkan fingerprint dengan host key VM yang sudah diverifikasi/panitia. Jika key benar, isi file group2-known-hosts akan dipakai untuk SSH_KNOWN_HOSTS. Workflow tetap mengaktifkan StrictHostKeyChecking; tidak menerima host key sembarangan.

## 6. Tambahkan GitHub Secrets

Buka repository → Settings → Secrets and variables → Actions → New repository secret.

| Name | Nilai |
|---|---|
| SSH_HOST | proxy.bccdev.id |
| SSH_PORT | Port SSH kelompok, contoh 11112 |
| SSH_USER | dev |
| SSH_PRIVATE_KEY | Seluruh isi private key group2, termasuk BEGIN/END |
| SSH_KNOWN_HOSTS | Seluruh isi file group2-known-hosts yang telah diverifikasi |

Nama secret harus persis. Nomor port adalah teks angka tanpa tanda kutip. Secret PRIVATE_KEY harus multiline asli, bukan public key dan bukan nama file. Isi GitHub Secrets melalui form GitHub, bukan file yang di-commit.

## 7. Salin paket ke repository Mac

Masuk ke folder repository lokal, lalu:

```bash
git status
```

Pastikan tidak ada merge/conflict atau perubahan kerja yang belum disimpan. Setelah bersih:

```bash
git switch main
git pull --ff-only origin main
```

Salin isi folder projecthub-deploy-files dari ZIP ke root repository dan replace file yang sama. Termasuk `.github/workflows/deploy.yaml`, Dockerfile, docker-compose.yaml, .dockerignore, deploy/deploy.sh, internal/middleware/cors.go dan internal/bootstrap/bootstrap.go. Di Finder, Cmd+Shift+. menampilkan file/folder tersembunyi. Jangan membuat folder bersarang tambahan di dalam repository.

File bootstrap pengganti dibuat dari commit main 2db7943. Jika bootstrap lokal sudah berubah sejak commit itu, gabungkan hanya perubahan CORS: membuat policy dari cfg.FrontendURL lalu router.Use(middleware.CORS(policy)) setelah gin.Default(). Jangan menghilangkan perubahan anggota tim yang lebih baru.

## 8. Periksa lalu push

```bash
gofmt -w internal/bootstrap/bootstrap.go internal/middleware/cors.go
go build ./...
go vet ./...
go test -race ./...
git diff --check
git status
```

Hasil `no test files` berarti package itu belum punya tes; bukan bukti alur database sudah diuji. Source main saat diperiksa belum membawa integration test dari ZIP perbaikan sebelumnya. Pipeline ini memeriksa tes yang memang ada, tidak menyatakan seluruh requirement aplikasi sudah selesai.

Jika tanpa error:

```bash
git add Dockerfile docker-compose.yaml .dockerignore .github/workflows/deploy.yaml deploy/deploy.sh internal/bootstrap/bootstrap.go internal/middleware/cors.go VM.env.example MULAI_CICD.md
git commit -m "ci: deploy backend to hackathon VM"
git push origin main
```

Jika main dilindungi, buat branch lalu ajukan PR ke main. Push/PR dev hanya menjalankan checks; deployment hanya berjalan pada main. Jika sudah ada workflow deployment lain di repository, nonaktifkan yang lama agar tidak menjalankan dua deployment bersamaan.

## 9. Pantau Actions

Buka tab Actions → Backend CI and VM Deployment. Urutan: checks (vet/test/build), build Docker image, siapkan SSH, upload image, deploy, tunggu health check. Tidak perlu registry Docker/GHCR karena image dikirim melalui SSH.

Migrasi yang gagal membuat proses server berhenti; health check gagal, job menjadi merah. Saat update, script mencoba mengembalikan image/Compose sebelumnya jika tersedia. Ini rollback aplikasi saja, bukan rollback perubahan schema database. Deployment pertama belum punya image lama untuk dikembalikan.

## 10. Periksa layanan

Dari Mac untuk kelompok 2:

```bash
curl -fsS https://group2-be.bccdev.id/health
```

Perkiraan respons:

```json
{"message":"Server is running"}
```

Di VM:

```bash
docker ps --filter name=projecthub-api
curl -fsS http://127.0.0.1:8080/health
docker logs --tail 80 projecthub-api
```

Jangan kirim log yang mengandung connection string/password. Jika localhost sehat tetapi domain gagal, cek routing domain panitia. Jika job gagal sebelum image terkirim, cek secrets/SSH. Jika unhealthy, cek log koneksi database/migrasi/env. Jangan melakukan reset/drop database untuk membuat deploy lolos.

Cek preflight dari Mac:

```bash
curl -i -X OPTIONS https://group2-be.bccdev.id/api/v1/projects \
  -H 'Origin: https://group2.bccdev.id' \
  -H 'Access-Control-Request-Method: GET' \
  -H 'Access-Control-Request-Headers: authorization'
```

Expected: HTTP 204 dan Access-Control-Allow-Origin sesuai domain frontend. Frontend memakai base URL `https://group2-be.bccdev.id/api/v1`.

## Scope validasi paket

YAML Compose/workflow dan bash deploy diperiksa secara lokal. Source main plus perubahan CORS diuji build/vet dan menjalankan go test sesuai isi source. Docker Engine, GitHub Actions, SSH/VM dan domain panitia belum dapat dijalankan di lingkungan ini. Panduan mengasumsikan kamu memiliki key VM yang benar dan akses repository untuk membuat secrets.

CI/CD ini baru bagian deployment backend. Stack monitoring yang diminta brief tetap perlu dipasang dan ditunjukkan; rotasi log container bukan pengganti monitoring. Modul aplikasi/realtime yang belum terhubung di main juga tidak otomatis diselesaikan oleh deployment.

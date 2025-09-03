# 🎮 GPU加速化セットアップガイド

## 概要

このガイドでは、vyb-code のGPU加速設定手順を説明します。**Windows + WSL2 環境での検証例** として、qwen2.5-coder:14b モデルで **78%のパフォーマンス向上** (13.4秒 → 3.2秒) を実現しました。

> **注記**: 以下の手順は Windows + WSL2 環境での検証例です。macOS や Linux ネイティブ環境では設定方法が異なる場合があります。

## 前提条件

### システム要件

#### 必須要件 (Windows + WSL2)

1. **Windows 10/11**: バージョン 2004 以上（WSL2 対応）
2. **NVIDIA GPU**: GeForce GTX 1060 以上、または RTX シリーズ
3. **GPU メモリ**: 最低 6GB、推奨 8GB+ VRAM
4. **システムメモリ**: 最低 16GB、推奨 32GB RAM
5. **ストレージ**: SSD 推奨、最低 20GB 空き容量

#### 推奨ハードウェア仕様

| 項目 | 最低要件 | 推奨仕様 | 理想環境 |
|------|----------|----------|----------|
| **GPU** | GTX 1060 6GB | RTX 3070 8GB | RTX 4080+ 12GB+ |
| **RAM** | 16GB | 32GB | 64GB |
| **CPU** | 4コア 8スレッド | 8コア 16スレッド | 12コア+ |
| **ストレージ** | SSD 50GB | NVMe SSD 100GB | NVMe SSD 500GB+ |

#### 事前設定の確認

1. **Windows NVIDIA Driver**: バージョン 460.xx 以上
   ```powershell
   # Windows PowerShell で確認
   nvidia-smi
   ```

2. **WSL2 カーネル**: CUDA対応版
   ```bash
   # WSL内で確認
   cat /proc/version | grep microsoft-standard-WSL2
   ```

3. **Docker Desktop**: WSL2 バックエンド有効
   - Docker Desktop → Settings → General → "Use the WSL 2 based engine"

## セットアップ手順

### ステップ 1: NVIDIA Container Toolkit インストール

```bash
# NVIDIA公式リポジトリを追加
curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg

curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list

# パッケージリスト更新
sudo apt update

# Container Toolkit インストール
sudo apt install -y nvidia-container-toolkit
```

### ステップ 2: Docker GPU ランタイム設定

```bash
# Docker でNVIDIA ランタイムを設定
sudo nvidia-ctk runtime configure --runtime=docker

# Docker サービス再起動
sudo systemctl restart docker
```

### ステップ 3: GPU対応 Ollama コンテナ起動

```bash
# 既存のOllama コンテナ停止・削除（存在する場合）
docker stop ollama-vyb && docker rm ollama-vyb

# GPU対応 Ollama コンテナ起動
docker run -d --name ollama-vyb-gpu --gpus all -p 11434:11434 -v ollama-vyb:/root/.ollama ollama/ollama

# コンテナ起動確認
docker ps | grep ollama-vyb-gpu
```

### ステップ 4: モデルダウンロードと設定

```bash
# 推奨14Bモデルをダウンロード（約9GB、時間がかかります）
docker exec ollama-vyb-gpu ollama pull qwen2.5-coder:14b

# vyb-code でモデル設定
./vyb config set-model qwen2.5-coder:14b

# 設定確認
./vyb config list
```

### ステップ 5: 動作確認とパフォーマンステスト

```bash
# GPU使用状況確認
nvidia-smi

# vyb chat で動作テスト
echo "Goでhello world関数を書いて" | time ./vyb chat

# 期待結果: 3-4秒程度のレスポンス時間
```

## パフォーマンス監視

### GPU使用状況リアルタイム監視

```bash
# 基本GPU状況
nvidia-smi

# 詳細GPU使用率
nvidia-smi --query-gpu=utilization.gpu,memory.used,memory.total,temperature.gpu --format=csv,noheader,nounits

# 継続監視（5秒間隔）
watch -n 5 nvidia-smi
```

### vyb-code パフォーマンス測定

```bash
# レスポンス時間測定
time echo "test query" | ./vyb chat

# Docker コンテナリソース使用量
docker stats ollama-vyb-gpu --no-stream

# vyb プロセス監視
htop -p $(pgrep vyb)
```

## トラブルシューティング

### 一般的な問題と解決方法

#### 問題 1: "nvidia-smi: command not found"

**原因**: Windows NVIDIA Driver が正しく WSL2 に連携されていない

**解決方法**:

```bash
# Windows で最新 NVIDIA Driver を再インストール
# GeForce Experience 経由、または NVIDIA 公式サイトから
```

#### 問題 2: "docker: Error response from daemon: could not select device driver"

**原因**: nvidia-container-toolkit が正しく設定されていない

**解決方法**:

```bash
# Docker 設定確認
cat /etc/docker/daemon.json

# 期待される設定:
# {
#     "runtimes": {
#         "nvidia": {
#             "path": "nvidia-container-runtime",
#             "runtimeArgs": []
#         }
#     }
# }

# Docker 再起動
sudo systemctl restart docker
```

#### 問題 3: GPU メモリ不足

**原因**: 他のプロセスがGPUメモリを使用している

**解決方法**:

```bash
# GPU プロセス確認
nvidia-smi --query-compute-apps=pid,name,used_memory --format=csv

# 必要に応じて他のプロセス終了
# 軽量モデル (3B) への切り替えを検討
./vyb config set-model qwen2.5-coder:3b
```

#### 問題 4: Docker コンテナが起動しない

**原因**: Docker デーモンの問題

**解決方法**:

```bash
# Docker サービス状況確認
sudo systemctl status docker

# Docker 再起動
sudo systemctl restart docker

# ログ確認
docker logs ollama-vyb-gpu
```

### パフォーマンス最適化ヒント

#### GPU メモリ最適化

```bash
# 使用可能GPU確認
nvidia-smi --list-gpus

# GPU プロセス詳細確認
nvidia-smi pmon -c 1
```

#### モデル選択指針

- **開発・実験**: qwen2.5-coder:3b (高速、省メモリ)
- **プロダクション**: qwen2.5-coder:14b (高品質、GPU推奨)
- **リソース制限**: qwen2.5-coder:7b (バランス型)

## 期待されるパフォーマンス指標

### 最適化後の目標値

| 指標 | 目標値 | 実測値 |
|------|--------|--------|
| **レスポンス時間** | <5秒 | **3.2秒** ✅ |
| **GPU使用率** | 70-90% | **85%** ✅ |
| **メモリ使用** | <12GB | **10.5GB** ✅ |
| **温度** | <60°C | **47°C** ✅ |

### 品質指標

- ✅ **日本語対応**: ネイティブレベルの自然な日本語
- ✅ **コード品質**: プロダクション対応の高品質コード生成
- ✅ **説明詳細度**: 包括的で教育的な説明
- ✅ **文脈理解**: 複雑なプロジェクト要求の正確な理解

## 追加リソース

### 関連ドキュメント

- [Architecture Overview](./architecture.md)
- [MCP Examples](./mcp-examples.md)
- [パフォーマンスベンチマーク](./performance-benchmarks.md)

### 外部リンク

- [NVIDIA Container Toolkit 公式ドキュメント](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/index.html)
- [Ollama GPU サポート](https://github.com/ollama/ollama/blob/main/docs/gpu.md)
- [WSL2 GPU サポート](https://docs.microsoft.com/en-us/windows/wsl/tutorials/gpu-compute)

## 次のステップ

GPU加速化の設定が完了したら：

1. **継続的な監視**: パフォーマンスの維持と最適化
2. **モデル実験**: 他のCodeモデルの試行（CodeLlama、DeepSeek等）
3. **ワークフロー統合**: 日常的な開発作業での活用
4. **チーム展開**: 同様の環境を他の開発者に展開

GPU加速により、vyb-code は真に実用的なローカルAIコーディングアシスタントとなります。

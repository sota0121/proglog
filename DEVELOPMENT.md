# Development Note

This document is intended to be a guide for developers who want to contribute to the project. It is not intended to be a guide for users who want to use the project.

## References

- This project is based on the book "Distributed Services with Go" by Travis Jeffery.
  - [en / Distributed Services with Go](https://pragprog.com/titles/tjgo/distributed-services-with-go/)
    - repo: https://github.com/travisjeffery/proglog (This is the original repo, but it is written in Go v1.13 style.)
  - [jp / Go言語による分散サービス](https://www.oreilly.co.jp/books/9784873119977/)
    - repo: https://github.com/YoshikiShibata/proglog (This is a forked repo, but it is written in Go v1.16 style. <-- recommended)


## Prerequisites

- Go v1.16 or later


## Install dependencies

- Install protoc
  - https://grpc.io/docs/protoc-installation/
- Install protobuff runtime for Go
  - https://developers.google.com/protocol-buffers/docs/gotutorial
  - `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`
  - `go install google.golang.org/protobuf/cmd/protoc-gen-go-grpc@latest`


## Third-party Go packages

See: [go.mod](./go.mod)

## Notes

### Part2 Section5 - Building a secure service

There are 3 steps to build a secure distributed service.

1. Encrypt the transported data
   1. 中間者攻撃から保護するために、通信データの暗号化を行う
2. Authentication
   1. クライアントを識別するために、認証を行う
3. Authorization
   1. 識別されたクライアントの権限を決定するために、認可を行う

#### Step1 - Encrypt the transported data

- Transporting data encryption prevents man-in-the-middle (MITM) attack.
- We need to use Secure Sockets Layer (SSL), and Transport Layer Security (TLS)
- TLS is the second gen of SSL


##### What is TLS ?

クライアントとサーバーの通信がTLSハンドシェイクによって開始される。ハンドシェイクは次のような手順を踏む。ハンドシェイクが完了すると、クライアントとサーバーは（MITM攻撃から）安全に通信することができるようになる。

1. 使われるTLSバージョンを指定
2. 使われる暗号スイート (cipher suite) を決める
3. サーバーの秘密鍵と認証局のデジタル署名により、サーバーの身元を確認（認証）する
4. ハンドシェイクが完了した後、対称鍵暗号のためのセッションキーを生成する

通常、TLSはライブラリが実行してくれる。したがって、我々アプリ開発者の仕事は、「証明書の取得」「証明書を使ってTLS通信するようgRPCに指示すること」である。このTLSにより、「クライアントがサーバーを認証する」作業が行われる。


#### Step2 - Authentication

このステップでは、「サーバーがクライアントを認証する」作業を行う。認証（Authentication）は、クライアントがサーバーに対して「自分は誰だ」ということを証明することである。

ほとんどのWebサービスでは、TLSを使って一方向認証、つまりサーバーの認証のみを行う。クライアントの認証はアプリケーションに任されていて、通常はユーザー名、パスワードの認証とトークンの組み合わせで行われる。

分散システムのようなマシン間の通信では、TLSを使って双方向認証を行う。サーバーとクライアントの両方が、自分の身元を証明するために証明書を使う。

この **相互TLS認証** は、効果的で比較的間簡単であり、広く採用されている。多くの企業が社内の分散サービス間の通信を安全にするために利用している。


#### Step3 - Authorization

認証と認可は密接に関連しているため、しばしば「 **auth** 」という言葉を使って両方を指すことがある。

認証と認可は、リクエストのライフサイクルでもサーバーのコード内でも基本的に **同時に行われる。**

ほとんどの基本的なWebサービス（Twitterなど）は、アカウントの所有者が一人であるため、認証・認可は同じ処理として扱われる。

ただし、様々なレベルの所有者とアクセスを共有しているリソースを持っている場合（Google Cloudなど）認証と認可は区別されなければならない。

たとえば、以下のように…

- Alice：アカウント所有者、ログの書き込み・読み出し権限あり
- Bob：ログの読み出し権限あり


上記のような細かいアクセスコントロールをする場合は、**認可が明示的に必要である。**

本リポジトリでは、 Access Control List (ACL) を使って認可を行う。


#### Authentication Server with TLS using CFSSL

社内サービスの場合、Certification Authority (CA) ベンダーなどは利用しなくていい。費用もかかるし、複雑になる。

代わりにCloud Flareがオープンソースで提供している **CFSSL** を使う。これにより、独自のCAとして運用できるようになる。

※実際CAベンダーも内部的にはCFSSLを使っている。

CFSSL has 2 main components:

1. **cfssl** - a command line tool for signing, verifying, and bundling TLS certificates. and exporting them in various formats including JSON.
2. **cfssljson** - a command line tool for splitting a JSON into key, certificate, CSR, and bundle files.


#### Step1-1 - Generate a Certificate Authority (CA)

Install Cloud Flare's CFSSL

```bash
go install github.com/cloudflare/cfssl/cmd/cfssl@latest
go install github.com/cloudflare/cfssl/cmd/cfssljson@latest
```

Make a directory for the CA

```bash
mkdir -p test
```

<br>

Create a CA configuration file

See: [`SecureYourService/test/ca-csr.json`](https://github.com/YoshikiShibata/proglog/blob/main/SecureYourServices/test/ca-csr.json)

- `CN` - Common Name
- `key` - Key type defines the algorithm and key size.
- `names` - The names list. Each name should have values below:
  - `C` - Country
  - `L` - Locality
  - `ST` - State or Province
  - `O` - Organization
  - `OU` - Organizational Unit


> **What is a CSR?**
> See: [CSRファイルとは何でしょうか？ | XTRUST](https://xtrust.jp/support/faq/faq08/a001/)

<br>

Create a CA policy file

See: [`SecureYourService/test/ca-config.json`](https://github.com/YoshikiShibata/proglog/blob/main/SecureYourServices/test/ca-config.json)

CA Policy file defines what kind of certificate can be published by the CA.

`signing` section defines the policy for signing certificates such as:

- expiration date (e.g. 8760h = 1 year)
- usages (e.g. server auth, client auth)

<br>

Create a CA CSR file for server

See: [`SecureYourService/test/server-csr.json`](https://github.com/YoshikiShibata/proglog/blob/main/SecureYourServices/test/server-csr.json)

- `hosts` - The hosts list. Each host should be a valid DNS name or IP address.
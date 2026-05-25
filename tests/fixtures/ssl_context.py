import ssl


def main() -> None:
    ctx = ssl.create_default_context()
    ctx.set_ciphers("HIGH")
    ctx.set_alpn_protocols(["h2", "http/1.1"])
    print("ok")


if __name__ == "__main__":
    main()

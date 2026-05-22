import ssl
import mmap


def main() -> None:
    print(ssl.CERT_NONE)
    print(ssl.CERT_REQUIRED)
    print(ssl.PROTOCOL_TLS_CLIENT)
    print(mmap.PROT_READ)
    print(mmap.MAP_SHARED)
    print(mmap.PAGESIZE > 0)


if __name__ == "__main__":
    main()

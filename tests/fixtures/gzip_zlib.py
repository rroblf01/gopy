import gzip
import zlib


def main() -> None:
    raw = "hello world hello world hello world"
    g = gzip.compress(raw.encode())
    back = gzip.decompress(g).decode()
    print(back == raw)
    print(len(g) > 0)
    z = zlib.compress(raw.encode())
    back2 = zlib.decompress(z).decode()
    print(back2 == raw)
    print(zlib.crc32(b"hello"))
    print(zlib.adler32(b"hello"))


if __name__ == "__main__":
    main()

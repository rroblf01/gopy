import tarfile
import zipfile
import lzma


def main() -> None:
    print(tarfile.PAX_FORMAT)
    print(tarfile.USTAR_FORMAT)
    print(zipfile.ZIP_STORED)
    print(zipfile.ZIP_DEFLATED)
    print(lzma.FORMAT_XZ)
    print(lzma.PRESET_DEFAULT)
    print(lzma.CHECK_CRC32)


if __name__ == "__main__":
    main()

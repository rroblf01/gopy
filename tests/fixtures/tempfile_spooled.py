import tempfile


def main() -> None:
    f = tempfile.SpooledTemporaryFile(mode="w+")
    f.write("hello")
    f.write(" world")
    f.seek(0)
    print(f.read())
    f.close()


if __name__ == "__main__":
    main()

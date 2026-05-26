import tempfile


def main() -> None:
    with tempfile.TemporaryFile(mode="w") as fh:
        fh.write("hello\n")
        fh.write("world\n")
    # Successful exit with no exception.
    print("ok")


if __name__ == "__main__":
    main()

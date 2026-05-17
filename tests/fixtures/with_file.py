def main() -> None:
    path: str = "/tmp/gopy_with_file_test.txt"
    with open(path, "w") as fh:
        fh.write("hello\nworld\n")
    with open(path, "r") as rh:
        content: str = rh.read()
        print(content)
        print(len(content))


if __name__ == "__main__":
    main()

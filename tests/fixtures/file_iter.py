def main() -> None:
    path: str = "/tmp/gopy_file_iter_test.txt"
    with open(path, "w") as fh:
        fh.write("alpha\nbeta\ngamma\n")
    count: int = 0
    with open(path, "r") as rh:
        for line in rh:
            count += 1
            # Python keeps the trailing newline from file iteration; the
            # Go shim's bufio.Scanner strips it. Normalize with strip()
            # so the fixture matches across runtimes.
            print(line.strip())
    print(count)


if __name__ == "__main__":
    main()

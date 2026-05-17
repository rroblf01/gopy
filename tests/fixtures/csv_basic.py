import csv


def main() -> None:
    # csv.reader accepts any iterable of lines; both CPython and the
    # gopy shim iterate to produce parsed rows.
    lines: list[str] = ["name,age", "ada,36", "grace,47"]
    rows: list[list[str]] = csv.reader(lines)
    for row in rows:
        for cell in row:
            print(cell)


if __name__ == "__main__":
    main()

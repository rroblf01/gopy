import csv


def main() -> None:
    path = "/tmp/gopy_csv_test.csv"
    with open(path, "w") as fh:
        w = csv.writer(fh)
        w.writerow(["a", "b", "c"])
        w.writerow(["1", "2", "3"])
        w.writerows([["4", "5", "6"], ["7", "8", "9"]])
    with open(path, "r") as fh:
        text = fh.read()
        print(text)


if __name__ == "__main__":
    main()

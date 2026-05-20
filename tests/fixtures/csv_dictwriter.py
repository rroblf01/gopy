import csv
from datetime import date, time, datetime
import getpass


def main() -> None:
    path = "/tmp/gopy_csv_dict.csv"
    with open(path, "w") as fh:
        w = csv.DictWriter(fh, ["name", "qty"])
        w.writeheader()
        w.writerow({"name": "ada", "qty": "3"})
        w.writerow({"name": "bob", "qty": "7"})
    with open(path, "r") as fh:
        print(fh.read())
    # datetime.combine
    d = date(2026, 5, 19)
    t = time(12, 34, 56)
    dt = datetime.combine(d, t)
    print(dt.strftime("%Y-%m-%d %H:%M:%S"))
    # getpass.getuser — non-empty value
    u = getpass.getuser()
    print(len(u) > 0)


if __name__ == "__main__":
    main()

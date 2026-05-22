import sqlite3
import doctest
import marshal


def main() -> None:
    print(sqlite3.PARSE_DECLTYPES)
    print(sqlite3.PARSE_COLNAMES)
    print(sqlite3.threadsafety)
    print(sqlite3.paramstyle)
    print(doctest.NORMALIZE_WHITESPACE)
    print(doctest.ELLIPSIS)
    print(doctest.SKIP)
    print(marshal.version)


if __name__ == "__main__":
    main()

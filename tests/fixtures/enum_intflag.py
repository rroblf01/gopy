from enum import IntFlag


class Perm(IntFlag):
    READ = 1
    WRITE = 2
    EXEC = 4


def main() -> None:
    p = Perm.READ | Perm.WRITE
    print(p == Perm.READ | Perm.WRITE)
    print(p & Perm.READ == Perm.READ)
    print(p & Perm.EXEC == Perm.EXEC)
    print((p | Perm.EXEC) == Perm.READ | Perm.WRITE | Perm.EXEC)
    print((Perm.READ ^ Perm.WRITE) == Perm.READ | Perm.WRITE)


if __name__ == "__main__":
    main()

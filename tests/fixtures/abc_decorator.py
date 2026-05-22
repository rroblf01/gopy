from abc import abstractmethod


class Base:
    @abstractmethod
    def m(self) -> int:
        return 0


class Sub(Base):
    def m(self) -> int:
        return 42


def main() -> None:
    s = Sub()
    print(s.m())


if __name__ == "__main__":
    main()

class Walker:
    def walk(self) -> None:
        print("walking")


class Talker:
    def talk(self) -> None:
        print("talking")


class Person(Walker, Talker):
    def greet(self) -> None:
        self.walk()
        self.talk()


def main() -> None:
    p = Person()
    p.greet()


if __name__ == "__main__":
    main()

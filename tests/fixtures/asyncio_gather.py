import asyncio


async def task(n: int) -> int:
    return n * 2


async def main() -> None:
    results = await asyncio.gather(task(1), task(2), task(3))
    total = 0
    for r in results:
        if isinstance(r, int):
            total += r
    print(total)


if __name__ == "__main__":
    asyncio.run(main())

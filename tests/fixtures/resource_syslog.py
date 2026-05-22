import resource
import syslog


def main() -> None:
    print(resource.RLIMIT_NOFILE)
    print(resource.RLIM_INFINITY)
    lim = resource.getrlimit(resource.RLIMIT_NOFILE)
    print(len(lim))
    print(syslog.LOG_INFO)
    print(syslog.LOG_DEBUG)
    syslog.openlog("app")
    syslog.syslog(syslog.LOG_INFO, "msg")
    syslog.closelog()
    print("ok")


if __name__ == "__main__":
    main()

export const MESSAGES = {
  validation: {
    usernameRequired: 'Введите имя пользователя',
    usernameMinLength: (min: number) =>
      `Имя пользователя должно быть минимум ${min} символа`,
    usernameMaxLength: (max: number) =>
      `Имя пользователя должно быть максимум ${max} символов`,
    usernameAllowedChars:
      'Имя пользователя может содержать только латинские буквы (A–Z), цифры, _ и -',
    usernameCannotStartOrEnd:
      'Имя пользователя не может начинаться или заканчиваться на _ или -',

    passwordRequired: 'Введите пароль',
    passwordMinLength: (min: number) =>
      `Пароль должен быть минимум ${min} символов`,
    passwordMaxLength: (max: number) =>
      `Пароль должен быть максимум ${max} символов`,
    passwordMustContainLatinLetter:
      'Пароль должен содержать хотя бы одну латинскую букву (A–Z)',
    passwordMustContainDigit: 'Пароль должен содержать хотя бы одну цифру',

    confirmPasswordRequired: 'Подтвердите пароль',
    passwordsDoNotMatch: 'Пароли не совпадают',

    messageRequired: 'Введите сообщение',
    messageTooLong: (max: number) =>
      `Сообщение слишком длинное (максимум ${max} символов)`,

    searchQueryRequired: 'Введите поисковый запрос',
    searchQueryTooLong: (max: number) =>
      `Поисковый запрос слишком длинный (максимум ${max} символов)`,
  },

  auth: {
    tabs: {
      login: 'Вход',
      register: 'Регистрация',
    },
    labels: {
      username: 'Имя пользователя',
      password: 'Пароль',
      confirmPassword: 'Подтвердите пароль',
    },
    actions: {
      show: 'Показать',
      hide: 'Скрыть',
      login: 'Войти',
      register: 'Зарегистрироваться',
      loggingIn: 'Входим...',
      registering: 'Регистрируем...',
    },
    toasts: {
      loginSuccess: 'Успешный вход',
      registerSuccess: 'Регистрация прошла успешно.',
      newKeyCreatedOnDevice:
        'Создан новый ключ шифрования для этого устройства.',
    },
    info: {
      usernameMin: (min: number) => `Минимум ${min} символа`,
      usernameMax: (max: number) => `Максимум ${max} символа`,
      usernameOk: '✓ Корректное имя',
      usernameAllowed: 'Только латинские буквы (A–Z), цифры, _ и -',
      noPasswordRecovery:
        'Восстановление пароля невозможно. Сохраните пароль в надёжном месте.',
      passwordsMatch: 'Пароли совпадают',
      passwordsMismatch: 'Пароли не совпадают',
    },
    passwordStrength: {
      min8: 'Минимум 8 символов',
      addUppercaseLatin: 'Добавьте заглавные латинские буквы (A–Z)',
      addDigits: 'Добавьте цифры',
      addSpecial: 'Добавьте спецсимволы (!@#$%...)',
      labels: {
        weak: 'Слабый',
        medium: 'Средний',
        strong: 'Сильный',
        veryStrong: 'Очень сильный',
      },
    },
    errors: {
      keygenDefault: 'Ошибка генерации ключей. Попробуйте ещё раз.',
    },
  },

  chat: {
    security: {
      identityVerifiedLabel: 'Identity подтверждён',
      identityVerifiedToast:
        'Identity подтверждён. Безопасное общение установлено.',
    },
    fingerprintModal: {
      title: 'Верификация Identity',
      aria: {
        close: 'Закрыть',
      },
      states: {
        loadingFingerprint: 'Загрузка отпечатка...',
      },
      warnings: {
        changedTitle: 'Внимание: отпечаток изменился!',
        changedBody:
          'Отпечаток этого пользователя отличается от сохранённого. Убедитесь, что вы общаетесь с правильным человеком.',
      },
      labels: {
        myFingerprint: 'Ваш отпечаток',
        notLoaded: 'Не загружен',
        visually: 'Визуально:',
        peerFingerprint: (peerUsername: string) => `Отпечаток ${peerUsername}`,
      },
      info: {
        instructionTitle: 'Инструкция:',
        instructionText:
          'Сравните эти отпечатки вне приложения (например, по телефону или в другом канале связи). Если они совпадают, нажмите "Подтвердить". Это защитит вас от атак типа man-in-the-middle.',
        tip: {
          emoji: '💡',
          title: 'Совет:',
          text: 'Используйте визуальные коды (эмодзи) для быстрого сравнения по телефону — они легче запоминаются и произносятся.',
        },
      },
      actions: {
        close: 'Закрыть',
        verify: 'Подтвердить',
      },
      errors: {
        failedToLoadFingerprint: 'Не удалось загрузить отпечаток',
      },
    },
    contextMenu: {
      actions: {
        copy: 'Копировать',
        react: 'Отреагировать',
        reply: 'Ответить',
        edit: 'Редактировать',
        deleteSelf: 'Удалить только у себя',
        deleteEveryone: 'Удалить у всех',
      },
    },
    messageBubble: {
      deleted: 'Сообщение удалено',
      you: 'Вы',
      peerFallback: 'Собеседник',
      edited: '(изменено)',
      actions: {
        save: 'Сохранить',
        cancel: 'Отмена',
      },
      replyPreview: {
        deleted: 'Сообщение удалено',
        voice: 'Голосовое сообщение',
        video: 'Видео сообщение',
        file: 'Файл',
        message: 'Сообщение',
      },
    },
    fileAccessDialog: {
      title: 'Выберите режим доступа',
      modes: {
        both: {
          title: 'Скачивание и просмотр',
          description: 'Собеседник сможет скачать и просмотреть файл',
        },
        viewOnly: {
          title: 'Только просмотр',
          description:
            'Собеседник сможет только просмотреть файл, но не скачать',
        },
        downloadOnly: {
          title: 'Только скачивание',
          description:
            'Собеседник сможет только скачать файл, но не просмотреть',
        },
      },
      actions: {
        cancel: 'Отмена',
        send: 'Отправить',
      },
    },
    fileMessage: {
      actions: {
        viewTitle: 'Просмотреть файл',
        view: 'Просмотреть',
        downloadTitle: 'Скачать файл',
        download: 'Скачать',
      },
    },
    fileViewerModal: {
      errors: {
        failedToLoadImage: 'Не удалось загрузить изображение',
        failedToReadFile: 'Не удалось прочитать файл',
        unsupportedType: 'Просмотр этого типа файла не поддерживается',
      },
      labels: {
        downloadFile: 'Скачать файл',
        preparingVideo: 'Подготовка видео...',
        videoLoadError: 'Ошибка загрузки видео',
        retry: 'Попробовать снова',
        videoNotSupported: 'Ваш браузер не поддерживает воспроизведение видео.',
        loading: 'Загрузка...',
        viewOnly: 'Только просмотр',
        download: 'Скачать',
      },
      aria: {
        close: 'Закрыть',
      },
    },
    fileTransfer: {
      errors: {
        missingChunks: (received: number, expected: number) =>
          `Не все части файла получены (${received}/${expected})`,
        emptyFile: 'Получен пустой файл',
        decryptFailed: 'Не удалось расшифровать файл',
        invalidTransfer: 'Получены некорректные данные файла',
        transferLimitExceeded:
          'Превышено количество одновременных передач файлов',
      },
    },
    incomingMessages: {
      errors: {
        decryptFailed:
          'Не удалось расшифровать сообщение. Возможно, сессия была прервана.',
        decryptEditedFailed:
          'Не удалось расшифровать отредактированное сообщение.',
      },
    },
    typingIndicator: {
      label: 'печатает...',
    },
    window: {
      labels: {
        securityCheck: 'Проверка безопасности...',
        secureSession: 'Защищённая сессия',
        replyToMessage: 'Ответ на сообщение',
      },
      banners: {
        securityCodesChanged: {
          title: 'Чат заблокирован: коды безопасности изменились!',
          description:
            'Верифицируйте Identity собеседника, чтобы продолжить общение.',
        },
      },
      emptyState: {
        title: 'Начните переписку',
        subtitle: 'Все сообщения защищены сквозным шифрованием',
      },
      aria: {
        closeChat: 'Закрыть чат',
        cancelReply: 'Отменить ответ',
        scrollToLastMessage: 'Прокрутить к последнему сообщению',
      },
      actions: {
        verifyIdentity: 'Проверить Identity',
        verified: 'Подтверждён',
        send: 'Отправить',
        attachFileTitle: 'Прикрепить файл',
      },
      placeholders: {
        blocked: 'Чат заблокирован: верифицируйте Identity',
        message: 'Введите сообщение...',
        waitingForSession: 'Ожидание установки сессии...',
      },
      status: {
        connecting: 'Подключение...',
        establishingSecureSession: 'Установка защищённой сессии...',
        peerOffline: 'Собеседник не в сети',
        peerDisconnected: 'Собеседник отключился',
        connectionError: 'Ошибка соединения',
      },
      reply: {
        voice: 'Голосовое сообщение',
        file: 'Файл',
        message: 'Сообщение',
      },
      warnings: {
        securityCodesChanged:
          'Коды безопасности изменились. Пожалуйста, верифицируйте Identity снова для безопасного общения.',
      },
      errors: {
        failedToLoadFingerprint:
          'Не удалось загрузить отпечаток. Попробуйте перезагрузить страницу.',
        failedToSendMessage:
          'Не удалось отправить сообщение. Проверьте соединение и попробуйте снова.',
        messageTooLong: (max: number) =>
          `Сообщение слишком длинное (максимум ${max} символов)`,
        failedToSendVoice:
          'Не удалось отправить голосовое сообщение. Проверьте микрофон и попробуйте снова.',
        failedToSendVideo:
          'Не удалось отправить видео сообщение. Проверьте камеру и попробуйте снова.',
        failedToSendFile:
          'Не удалось отправить файл. Проверьте соединение и попробуйте снова.',
      },
      toasts: {
        voiceSent: 'Голосовое сообщение отправлено',
        videoSent: 'Видео сообщение отправлено',
        fileSent: 'Файл отправлен',
      },
    },
    session: {
      errors: {
        failedToEstablishSession:
          'Не удалось установить сессию. Переподключитесь',
        signatureCheckFailed: 'Ошибка проверки подписи. Переподключитесь',
        failedToEstablishSecureSession:
          'Не удалось установить защищенную сессию. Попробуйте переподключиться.',
        peerOffline: 'Собеседник не в сети',
        identityKeyNotFound:
          'Ключ идентификации не найден. Перезайдите в систему',
        sessionSetupErrorTitle: 'Ошибка установки сессии',
        failedToSendMessage: 'Не удалось отправить сообщение',
        sessionNotActive: 'Сессия не активна. Подождите подключения',
        fileValidationError: 'Ошибка валидации файла',
        unknownError: 'неизвестная ошибка',
        fileEncryptionError: (reason: string) =>
          `Ошибка шифрования файла: ${reason}. Попробуйте выбрать другой файл.`,
        fileSendError: 'Ошибка отправки файла',
        voiceValidationError: 'Ошибка валидации голосового сообщения',
        editTimeLimit: 'Редактирование доступно только 15 минут после отправки',
        failedToEditMessage: 'Не удалось отредактировать сообщение',
        connectionError: 'Ошибка соединения',
      },
    },
    videoRecorderModal: {
      errors: {
        noCameraAccess: 'Не удалось получить доступ к камере',
        recordingError: 'Ошибка при записи видео',
        failedToStartRecording: 'Не удалось начать запись видео',
      },
      labels: {
        initializingCamera: 'Инициализация камеры...',
        readyTitle: 'Готов к записи',
        readyHint: 'Нажмите ▶ для начала',
        close: 'Закрыть',
        file: 'Файл',
        videoFile: (mb: string) => `Видео (${mb} MB)`,
        videoTooLarge: (mb: string) => `Видео слишком большое (${mb} MB)`,
      },
      titles: {
        start: 'Начать запись',
        cancel: 'Отменить',
        stop: 'Остановить запись',
        cancelRecording: 'Отменить запись',
      },
    },
    videoRecorder: {
      errors: {
        notSupported: 'Запись видео не поддерживается',
      },
      titles: {
        record: 'Записать видео сообщение',
      },
    },
    videoCircle: {
      aria: {
        play: (filename: string) => `Воспроизвести видео: ${filename}`,
      },
    },
    voiceRecorder: {
      errors: {
        notSupported: 'Запись аудио не поддерживается',
        notRecorded: 'Голосовое сообщение не было записано. Попробуйте снова',
        emptyRecording:
          'Записанное голосовое сообщение пустое. Попробуйте снова',
        stopError: 'Ошибка при остановке записи',
        startError: 'Не удалось начать запись. Проверьте разрешения микрофона.',
      },
      labels: {
        cancel: 'Отмена',
      },
      titles: {
        stop: 'Остановить запись',
        cancel: 'Отменить запись',
        record: 'Записать голосовое сообщение',
      },
    },
  },

  crypto: {
    signature: {
      errors: {
        signEphemeralKeyFailed: (details: string) =>
          `Не удалось подписать ephemeral key: ${details}. Ключ идентификации может быть недействителен или повреждён.`,
      },
    },
    identity: {
      errors: {
        keygenNotSupported:
          'Генерация ключей не поддерживается. Используйте современный браузер (Chrome, Firefox, Safari, Edge).',
        keygenRequiresHttps:
          'Для генерации ключей требуется безопасное соединение (HTTPS).',
        keygenFailed: (details: string) =>
          `Ошибка генерации ключей: ${details}`,
      },
    },
  },

  common: {
    words: {
      errorLower: 'ошибка',
      reconnect: 'переподключ',
    },
    http: {
      errorWithStatus: (status: number) => `Ошибка ${status}`,
    },
    browserSupport: {
      unsupportedFeatures: (features: string) =>
        `Ваш браузер не поддерживает: ${features}. Используйте современный браузер (Chrome, Firefox, Safari, Edge).`,
      webCryptoNotSupported: 'Web Crypto API не поддерживается',
      audioRecordingNotSupported: 'Запись аудио не поддерживается',
    },
    websocket: {
      tokenRefreshFailed: 'Не удалось обновить токен доступа. Войдите снова.',
      tokenRefreshError: 'Ошибка при обновлении токена доступа. Войдите снова.',
      tokenExpiredRefreshing: 'Токен доступа истек. Попытка обновления...',
      messageHandlingError: 'Ошибка обработки сообщения',
      connectionError: 'Ошибка подключения',
      failedToConnect: 'Не удалось подключиться',
      failedToEstablishConnection: 'Не удалось установить соединение',
      notConnected: 'WebSocket не подключен',
    },
    audioRecorder: {
      errors: {
        alreadyStarted: 'Запись уже начата',
        startTimeout: 'Таймаут запуска записи',
        mediaRecorderStartError: 'Ошибка MediaRecorder при запуске',
        failedToStart: (details: string) =>
          `Не удалось начать запись: ${details}`,
      },
    },
    errorBoundary: {
      confirmReload:
        'Обнаружена устаревшая версия приложения. Перезагрузить страницу?',
      staleTitle: 'Устаревшая версия приложения',
      errorTitle: 'Произошла ошибка',
      staleDescription:
        'Обнаружена устаревшая версия приложения. Пожалуйста, перезагрузите страницу.',
      unknownError: 'Неизвестная ошибка',
      actions: {
        reload: 'Перезагрузить страницу',
        retry: 'Попробовать снова',
      },
    },
    loading: {
      default: 'Загрузка...',
    },
    main: {
      confirmReload:
        'Обнаружена устаревшая версия приложения. Перезагрузить страницу?',
    },
    toastProvider: {
      aria: {
        close: 'Закрыть',
      },
    },
  },

  app: {
    authScreen: {
      description:
        'End-to-end зашифрованный чат 1-на-1. Начните с регистрации или входа.',
    },
    chatScreen: {
      subtitle: 'Защищённый мессенджер с E2E шифрованием и DH‑обменом ключами',
      actions: {
        logout: 'Выйти',
        search: 'Поиск',
        clearSearchAria: 'Очистить поиск',
      },
      profile: {
        title: 'Профиль',
        name: 'Имя:',
        choosePeer: 'Выберите собеседника, чтобы начать защищённый диалог.',
        myFingerprintButton: 'Мой отпечаток',
        myFingerprintAria: 'Показать мой отпечаток',
      },
      search: {
        title: 'Найти собеседника',
        placeholder: 'Имя пользователя...',
        userNotFound: 'Пользователь не найден',
        userNotFoundHint:
          'Попробуйте изменить запрос или проверьте правильность написания',
        enterUsernameToSearch: 'Введите имя пользователя для поиска',
      },
    },
    errors: {
      failedToLoadProfile: 'Не удалось получить профиль. Войдите снова',
    },
  },

  apiErrors: {
    invalidCredentials: 'Неверное имя пользователя или пароль',
    usernameTaken: 'Имя пользователя уже занято',
    usernameBetween: 'Имя пользователя должно быть от 3 до 32 символов',
    passwordBetween: 'Пароль должен быть от 8 до 72 символов',
    usernameAllowedChars:
      'Имя пользователя может содержать только латинские буквы (A–Z), цифры, подчёркивание и дефис',
    passwordMustContainLatinAndDigit:
      'Пароль должен содержать хотя бы одну латинскую букву (A–Z) и одну цифру',
    queryEmpty: 'Поисковый запрос не может быть пустым',
    queryTooLong: 'Поисковый запрос слишком длинный',
    sessionExpired: 'Сессия истекла. Войдите снова',
    serviceUnavailable: 'Сервис временно недоступен',
    fileTooLarge: 'Файл слишком большой (макс. 50MB)',
    userNotFound: 'Пользователь не найден',
    identityKeyNotFound: 'Ключ идентификации не найден',
    failedToLoadProfile: 'Не удалось загрузить профиль',
    searchFailed: 'Ошибка поиска пользователей',
    failedToGetIdentityKey: 'Не удалось получить ключ идентификации',
    failedToGetFingerprint: 'Не удалось получить отпечаток',
    offline: 'Нет подключения к интернету',
    noServerConnection: 'Нет соединения с сервером',
    timeout: 'Превышено время ожидания',
    connectionError: 'Ошибка подключения',
    reconnecting: 'Переподключение...',
    encryptionError: 'Ошибка шифрования',
    fileEncryptError: 'Не удалось зашифровать файл',
    fileDecryptError: 'Не удалось расшифровать файл',
    generalError: 'Произошла ошибка',
    badRequest: 'Неверный запрос',
    forbidden: 'Доступ запрещён',
    notFound: 'Не найдено',
    tooManyRequests: 'Слишком много запросов. Подождите',
    serverError: 'Ошибка сервера. Попробуйте позже',
  },
} as const;

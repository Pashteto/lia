package html

const VerificationEmailTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8" />
    <title>Подтверждение почты</title>
    <style>
        body { font-family: Arial, sans-serif; padding: 20px; background-color: #f4f4f4; }
        p { margin: 8px 0; }
        .container { background:#fff; padding:24px; border-radius:8px; max-width:600px; margin:0 auto; line-height:1.5; }
        .code { font-size:32px; letter-spacing:8px; font-weight:700; color:#111; margin:16px 0; }
        .muted { color:#666; font-size:13px; }
    </style>
</head>
<body>
<div class="container">
    <h2>Подтвердите вашу электронную почту</h2>
    <p>Введите этот код, чтобы подтвердить адрес электронной почты:</p>
    <div class="code">{{ .Code }}</div>
    <p class="muted">Код действителен 15 минут. Если вы не запрашивали подтверждение, просто проигнорируйте это письмо.</p>
    <p>С уважением,<br/>Команда Presence</p>
</div>
</body>
</html>`

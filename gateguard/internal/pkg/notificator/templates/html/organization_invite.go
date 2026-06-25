package html

const EmailTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8" />
    <title>Email Notification</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            padding: 20px;
            background-color: #f4f4f4;
        }
        p {
            margin: 8px 0;
        }
        .container {
            background-color: #ffffff;
            padding: 20px;
            border-radius: 5px;
            box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
            max-width: 600px;
            margin: 0 auto;
            line-height: 1.5;
        }
        .btn {
            display: inline-block;
            padding: 10px 20px;
            background-color: rgb(137, 80, 250);
            color: #ffffff !important;
            text-decoration: none;
            border-radius: 20px; /* Updated border-radius */
            transition: background-color 0.3s;
            margin: 10px 0;
        }
        .btn:hover {
            background-color: #3498db; /* Lighter blue on hover */
        }
    </style>
</head>
<body>
<div class="container">
    <h2>You have an invitation to a Presto Organization!</h2>
    <p>
        User "{{ .InviterEmail}}" has invited you to join their organization "{{ .OrganizationName}}".
    </p>
    <a href="{{ .ReferralLink}}" class="btn">Join Organization</a>

    <p>
        Presto Organizations allow you to collaborate with your team on managing rollups efficiently.
    </p>
    <p>
        Simply follow the link above and sign in using your preferred authentication method to get
        started.
    </p>

    <p>
        If you have any questions, suggestions, or feedback, please feel free to reply to this email.
        We're here to help!
    </p>
    <p>Best regards,</p>
    <p><strong>Presto Bot</strong></p>
</div>
</body>
</html>`

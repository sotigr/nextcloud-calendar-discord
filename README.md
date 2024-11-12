## Docker compose example

```yml
services:
  discordtasks:
    container_name: discordtasks
    image: sotig/nextcloudcaldiscord:latest 
    environment: 
      - NEXTCLOUD_SERVER=https://nextcloud.example.org
      - NEXTCLOUD_USER=myuser
      - NEXTCLOUD_PASSWORD=mypassword
      - NEXTCLOUD_CALENDARS=mycalendar|mywebhook
      - WEBHOOKS=mywebhook|https://discord.com/api/webhooks/...
      - RETRIEVAL_INTERVAL_MINUTES=10
```
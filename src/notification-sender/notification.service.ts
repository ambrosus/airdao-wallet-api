import * as admin from "firebase-admin";


export class NotificationService  {
    private readonly fcmClient: admin.messaging.Messaging;
    private readonly androidChannel: string;

    constructor(fcmClient: admin.messaging.Messaging, androidChannel: string) {
        this.fcmClient = fcmClient;
        this.androidChannel = androidChannel;
    }

    async sendNotification(
        {title, body, pushToken, data}: {title: string, body: string, pushToken: string, data: Record<string, unknown>}
    ): Promise<string | null> {
        const androidData: Record<string, string> = {};
        Object.entries(data).forEach(([key, value]) => {
            switch (typeof value) {
                case "string":
                    androidData[key] = value;
                    break;
                case "number":
                    androidData[key] = value.toString();
                    break;
                case "boolean":
                    androidData[key] = value.toString();
                    break;
                case "object":
                    androidData[key] = JSON.stringify(value);
                    break;
            }
        });

        console.log("AndroidData:", androidData);
        console.log("IOSData:", data);

        const message = {
            notification: {
                title: title,
                body: body,
            },
            android: {
                notification: {
                    title: title,
                    body: body,
                    channelId: this.androidChannel,
                },
                data: androidData,
            },
            token: pushToken,
        };

        try {
            const response = await this.fcmClient.send(message);
            return response;
        } catch (error) {
            console.error("Error sending message:", error);
            return null;
        }
    }
}

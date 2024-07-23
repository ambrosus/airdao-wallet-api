import { singleton } from "tsyringe";
import * as admin from "firebase-admin";

interface Notification {
    title: string;
    body: string;
    pushToken: string;
    data: Record<string, unknown>;
}

@singleton()
export class NotificationService  {
    constructor(
        private readonly fcmClient: admin.messaging.Messaging,
        private readonly androidChannel: string
    ) {}

    async sendNotification({
            title,
            body,
            pushToken,
            data
        }: Notification ): Promise<string | null> {
        console.log("ANDROID CHANNEL", this.androidChannel);
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

        console.log("androidData", androidData);

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

        console.log("SENDING NOTIFICATION", message);

        try {
            const response = await this.fcmClient.send(message);
            return response;
        } catch (error) {
            console.error("Error sending message:", { error, pushToken, message });
            return null;
        }
    }
}

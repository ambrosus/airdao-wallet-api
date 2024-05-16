import { Messaging, getMessaging } from 'firebase/messaging';
import { FirebaseApp } from 'firebase/app';

interface INotificationSenderService {
    sendMessage(title: string, body: string, pushToken: string, data: Record<string, any>): Promise<string | null>;
}

class NotificationSenderService implements INotificationSenderService {
    private fcmClient: Messaging;
    private androidChannel: string;

    constructor(firebaseApp: FirebaseApp, androidChannel: string) {
        this.fcmClient = getMessaging(firebaseApp);
        this.androidChannel = androidChannel;
    }

    async sendMessage(title: string, body: string, pushToken: string, data: Record<string, any>): Promise<string | null> {
        const androidData: Record<string, string> = {};
        Object.entries(data).forEach(([key, value]) => {
            switch (typeof value) {
                case 'string':
                    androidData[key] = value;
                    break;
                case 'number':
                    androidData[key] = value.toString();
                    break;
                case 'boolean':
                    androidData[key] = value.toString();
                    break;
                case 'object':
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
                    sound: 'default',
                },
                data: androidData,
            },
            token: pushToken,
        };

        try {
            // const response = await this.fcmClient.send(message);
            // return response;
            // todo: change me to response
            return "change me to response"
        } catch (error) {
            console.error("Error sending message:", error);
            return null;
        }
    }
}

interface MessageLog {
    [pushToken: string]: {
        title: string;
        body: string;
        data: Record<string, any>;
    }[];
}

class DevNotificationSenderService implements INotificationSenderService {
    private messageLog: MessageLog = {};

    async sendMessage(title: string, body: string, pushToken: string, data: Record<string, any>): Promise<string | null> {
        // Log the message
        if (!this.messageLog[pushToken]) {
            this.messageLog[pushToken] = [];
        }
        this.messageLog[pushToken].push({ title, body, data });

        // Return a dummy message ID
        return "dummy_message_id";
    }

    // For testing purposes, you can expose a method to retrieve the message log
    getMessageLog(pushToken: string): { title: string; body: string; data: Record<string, any>; }[] {
        return this.messageLog[pushToken] || [];
    }
}

// // Usage
// async function testMockCloudMessagingService() {
//     const mockCloudMessagingService = new MockCloudMessagingService();
//
//     await mockCloudMessagingService.sendMessage("Test Title 1", "Test Body 1", "pushToken1", { key: "value1" });
//     await mockCloudMessagingService.sendMessage("Test Title 2", "Test Body 2", "pushToken2", { key: "value2" });
//
//     // Retrieve message log for a specific device
//     const messagesForPushToken1 = mockCloudMessagingService.getMessageLog("pushToken1");
//     console.log("Messages sent to pushToken1:", messagesForPushToken1);
//
//     const messagesForPushToken2 = mockCloudMessagingService.getMessageLog("pushToken2");
//     console.log("Messages sent to pushToken2:", messagesForPushToken2);
// }
//
// testMockCloudMessagingService();

//
// // Usage
// async function testCloudMessagingService() {
//     const firebaseApp = getApp(); // Assuming you have a function to initialize Firebase app
//     const androidChannel = "your_android_channel_id";
//     const cloudMessagingService = new CloudMessagingServiceImpl(firebaseApp, androidChannel);
//
//     try {
//         const response = await cloudMessagingService.sendMessage("Test Title", "Test Body", "pushToken", { key: "value" });
//         console.log("Message sent, response:", response);
//     } catch (error) {
//         console.error("Error sending message:", error);
//     }
// }
//
// testCloudMessagingService();

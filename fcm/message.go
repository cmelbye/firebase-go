package fcm

import (
	"encoding/json"
	"time"
)

// Message represents a single message to send to one or more clients
// using FCM.
//
// For more information, see the documentation at:
// https://firebase.google.com/docs/cloud-messaging/http-server-ref#downstream-http-messages-json
type Message struct {
	// To specifies the recipient of a message.
	//
	// The value can be a device's registration token, a device group's
	// notification key, or a single topic (prefixed with "/topics/").
	// To send to multiple topics, use the Condition field.
	To string `json:"to,omitempty"`

	// RegistrationIDs specifies the recipient of a multicast message:
	// a message sent to more than one registration token.
	//
	// The value should be an array of registration tokens to which to
	// send the multicast message. The array must contain at least 1 and
	// at most 1000 registration tokens. To send a message to a single
	// device, use the To field.
	RegistrationIDs []string `json:"registration_ids,omitempty"`

	// Condition specifies a logical expression of conditions that determine
	// the message target.
	//
	// Supported condition: Topic, formatted as "'yourTopic' in topics".
	// This value is case-insensitive.
	//
	// Supported operators: &&, ||. Maximum two operators per topic message supported.
	Condition string `json:"condition,omitempty"`

	// CollapseKey identifies a group of messages (e.g., with CollapseKey: "Updates Available")
	// that can be collapsed, so that only the last message gets sent when delivery
	// can be resumed. This is intended to avoid sending too many of the same
	// messages when the device comes back online or becomes active.
	//
	// Note that there is no guarantee of the order in which messages get sent.
	//
	// Note: A maximum of 4 different collapse keys is allowed at any given time.
	// This means a FCM connection server can simultaneously store 4 different
	// send-to-sync messages per client app. If you exceed this number, there is
	// no guarantee which 4 collapse keys the FCM connection server will keep.
	CollapseKey string `json:"collapseKey,omitempty"`

	// Priority sets the priority of the message. Use the NormalPriority or
	// HighPriority constants to specify a priority.
	// On iOS, these correspond to APNs priorities 5 and 10.
	//
	// By default, notification messages are sent with high priority, and data
	// messages are sent with normal priority. Normal priority optimizes the
	// client app's battery consumption and should be used unless immediate
	// delivery is required. For messages with normal priority, the app may
	// receive the message with unspecified delay.
	//
	// When a message is sent with high priority, it is sent immediately,
	// and the app can wake a sleeping device and open a network connection
	// to your server.
	//
	// For more information, see the documentation at:
	// https://firebase.google.com/docs/cloud-messaging/concept-options#setting-the-priority-of-a-message
	Priority Priority `json:"priority,omitempty"`

	// ContentAvailable is used by iOS to represent content-available in the
	// APNs payload. When a notification or message is sent and this is set
	// to true, an inactive client app is awoken. On Android, data messages
	// wake the app by default. On Chrome, this is currently not supported.
	ContentAvailable bool `json:"content_available,omitempty"`

	// MutableContent is currently for iOS 10+ devices only. On iOS,
	// use this field to represent mutable-content in the APNS payload.
	// When a notification is sent and this is set to true, the content
	// of the notification can be modified before it is displayed, using a
	// Notification Service app extension. This parameter will be ignored
	// for Android and web.
	//
	// For more information about Notification Service app extensions, see the
	// documentation at: https://developer.apple.com/reference/usernotifications/unnotificationserviceextension
	MutableContent bool `json:"mutable_content,omitempty"`

	// TimeToLive specifies how long (in seconds) the message should be kept
	// in FCM storage if the device is offline. The maximum time to live
	// supported is 4 weeks, and the default value is 4 weeks.
	//
	// For more information, see the ocumentation at:
	// https://firebase.google.com/docs/cloud-messaging/concept-options#ttl
	TimeToLive int `json:"time_to_live,omitempty"`

	// RestrictedPackageName specifies the package name of the application
	// where the registration tokens must match in order to receive the message.
	RestrictedPackageName string `json:"restricted_package_name,omitempty"`

	// DryRun allows developers to test a request without actually sending a message.
	DryRun bool `json:"dry_run,omitempty"`

	// Data specifies the custom key-value pairs of the message's payload.
	//
	// For example, with Data: map[string]string{"score": "3x1"}:
	//
	// On iOS, if the message is sent via APNS, it represents the custom data fields.
	// If it is sent via FCM connection server, it would be represented as key
	// value dictionary in AppDelegate application:didReceiveRemoteNotification:.
	//
	// On Android, this would result in an intent extra named "score" with the
	// string value "3x1".
	//
	// The key should not be a reserved word ("from" or any word starting
	// with "google" or "gcm"). Do not use any of the words defined as part of the
	// FCM message protocol (see https://firebase.google.com/docs/cloud-messaging/http-server-ref).
	Data map[string]string `json:"data,omitempty"`

	// Notification specifies the predefined, user-visible key-value pairs of the
	// notification payload. See the Notification type for more information.
	//
	// For more information about notification message and data message options, see
	// the documentation at: https://firebase.google.com/docs/cloud-messaging/concept-options#notifications_and_data_messages
	Notification *Notification `json:"notification,omitempty"`
}

type Notification struct {
	// Title is the notification's title.
	//
	// For iOS, it is not visible on iOS phones and tablets.
	Title string `json:"title,omitempty"`

	// Body is the notification's body text.
	Body string `json:"body,omitempty"`

	// The sound to play when the device receives the notification.
	//
	// For iOS, sound files can be in the main bundle of the client app or in
	// the Library/Sounds folder of the app's data container.
	// See the iOS Developer Library for more information:
	// https://developer.apple.com/library/content/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/SupportingNotificationsinYourApp.html#//apple_ref/doc/uid/TP40008194-CH4-SW10
	//
	// For Android, this supports "default" or the filename of a sound resource
	// bundled in the app. Sound files must reside in "/res/raw/".
	Sound string `json:"sound,omitempty"`

	// Icon is used on Android to set the notification's icon.
	//
	// It sets the notification icon to the drawable resource given as a string.
	// If you don't send this key in the request, FCM displays the launcher icon specified in your app manifest.
	//
	// For iOS, this is unused.
	Icon string `json:"icon,omitempty"`

	// Badge is used on iOS to set the value of the badge on the home screen app icon.
	//
	// If not specified, the badge is not changed.
	//
	// If set to "0", the badge is removed.
	//
	// For Android, this is unused.
	Badge string `json:"badge,omitempty"`

	// Tag is used on Android to replace existing notifications in the notification drawer.
	//
	// If not specified, each request creates a new notification.
	//
	// If specified and a notification with the same tag is already being shown, the new notification replaces the existing one in the notification drawer.
	Tag string `json:"tag,omitempty"`

	// Color is used on Android to set the notification's icon color, expressed in #rrggbb format.
	//
	// For iOS, this is unused.
	Color string `json:"color,omitempty"`

	// ClickAction is the action associated with a user click on the notification.
	//
	// For iOS, this corresponds to category in the APNs payload.
	//
	// For Android, if this is specified, an activity with a matching intent filter
	// is launched when a user clicks on the notification.
	ClickAction string `json:"click_action,omitempty"`

	// TitleLocKey is the key to the title string in the app's string resources
	// to use to localize the title text to the user's current localization.
	//
	// For iOS, this corresponds to title-loc-key in the APNs payload.
	// See Payload Key Reference and Localizing the Content of Your Remote Notifications
	// for more information, at
	// https://developer.apple.com/library/content/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/PayloadKeyReference.html
	// and
	// https://developer.apple.com/library/content/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/CreatingtheNotificationPayload.html#//apple_ref/doc/uid/TP40008194-CH10-SW9
	// respectively.
	//
	// For Android, see the documentation on String Resources for more information:
	// https://developer.android.com/guide/topics/resources/string-resource.html
	TitleLocKey string `json:"title_loc_key,omitempty"`

	// TitleLocArgs contains string values to be used in place of the format
	// specifiers in TitleLocKey to use to localize the title text to the
	// user's current localization.
	//
	// For iOS, this corresponds to title-loc-args in the APNs payload.
	//
	// For Android, see the documentation on Formatting and Styling
	// for more information: https://developer.android.com/guide/topics/resources/string-resource.html#FormattingAndStyling
	TitleLocArgs StringArgs `json:"title_loc_args,omitempty"`

	// BodyLocKey is the key to the body string in the app's string resources
	// to use to localize the body text to the user's current localization.
	//
	// For iOS, this corresponds to loc-key in the APNs payload.
	BodyLocKey string `json:"body_loc_key,omitempty"`

	// BodyLocArgs contains string values to be used in place of the format
	// specifiers in BodyLocKey to use to localize the body text to the
	// user's current localization.
	//
	// For iOS, this corresponds to loc-args in the APNs payload.
	BodyLocArgs StringArgs `json:"body_loc_args,omitempty"`
}

// Priority is the message priority.
type Priority string

const (
	HighPriority   Priority = "high"
	NormalPriority Priority = "normal"
)

func (p Priority) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(p))
}

// StringArgs is a list of strings that, when JSON-marshalled, becomes
// the string encoding of a JSON array, rather than the JSON array itself.
//
// For example, {"foo", "bar"} becomes `["foo", "bar"]` rather than ["foo", "bar"].
//
// This is necessary for some fields due to the peculiarity of the FCM protocol.
type StringArgs []string

func (args StringArgs) MarshalJSON() ([]byte, error) {
	// Marshal to a regular JSON array
	bytes, err := json.Marshal([]string(args))
	if err != nil {
		return nil, err
	}

	// Then marshal that once more
	return json.Marshal(string(bytes))
}

// Response describes the result of sending one or more messages,
// as received by the FCM server.
///
// For more information see the documentation at:
// https://firebase.google.com/docs/cloud-messaging/http-server-ref#interpret-downstream
type Response struct {
	// MulticastID is the unique ID identifying the multicast message.
	MulticastID int64 `json:"multicast_id"`

	// Success is the number of messages that were processed without an error.
	Success int `json:"success"`

	// Failure is the number of messages that could not be processed.
	Failure int `json:"failure"`

	// CanonicalIDs is the number of results that contain a canonical
	// registration token. A canonical registration ID is the registration
	// token of the last registration requested by the client app.
	// This is the ID that the server should use when sending messages to the device.
	CanonicalIDs int `json:"canonical_ids"`

	// Results is an array of objects representing the status of the messages
	// processed. The objects are listed in the same order as the request
	// (i.e., for each registration ID in the request, its result is listed in
	// the same index in the response).
	Results []MessageResult `json:"results"`

	// RetryAfter indicates when the request should be retried.
	// It is the zero value if no such hint was given.
	RetryAfter time.Duration
}

// MessageResult describes the result of sending a message to a single device.
// See the Response type's Results field for more information.
type MessageResult struct {
	// MessageID is a unique ID for each successfully processed message.
	// It is the empty string if and only if there is an error.
	MessageID string `json:"message_id"`

	// RegistrationID specifies the canonical registration token for the
	// client app that the message was processed and sent to.
	// The sender should use this value as the registration token for
	// future requests. Otherwise, the messages might be rejected.
	//
	// If the sender is already using the canonical registration token,
	// the field is empty.
	RegistrationID string `json:"registration_id"`

	// Error specifies the error that occurred when processing the message
	// for the recipient. The empty string indicates no error.
	//
	// For possible error values, see the documentation at:
	// https://firebase.google.com/docs/cloud-messaging/http-server-ref#table9
	Error string `json:"error"`
}

export function setDefaultSetting(kiwiApi, settingKey, value) {
    const fullKey = `settings.${settingKey}`;
    if (kiwiApi.state.getSetting(fullKey) === undefined) {
        kiwiApi.state.setSetting(fullKey, value);
    }
}

// @ts-nocheck -- Generated dynamic contracts from Dialog.aui; edit the AUI source.
import {
  Button,
  Card,
  Heading,
  Input,
  Page,
  Row,
  Section,
  Spacer,
  Text,
} from "@/components/ui";

export function Dialog({
  data,
  onAction,
}: {
  data: any;
  onAction: (name: string) => void;
}) {
  return (
    <Page>
      <Section region="modal-backdrop">
        <Card region="modal">
          <Row>
            <Heading level={2}>{data.app.dialogTitle}</Heading>
            <Spacer />
            <Button
              intent="close-dialog"
              variant="icon"
              title="Close dialog"
              icon="X"
            />
          </Row>
          {data.app.dialogMessage && (
            <Text tone="muted">{data.app.dialogMessage}</Text>
          )}
          {data.app.dialogFields.map((item, i) => (
            <Input
              key={i}
              field={item.key}
              kind={item.kind}
              value={item.value}
              title={item.label}
              options={item.options}
              placeholder={item.placeholder}
              accept={item.accept}
            />
          ))}
          {data.app.dialogActions.map((item, i) => (
            <Button
              key={i}
              intent={item.intent}
              payload={item.payload}
              variant={item.variant}
              icon={item.icon}
            >
              {item.label}
            </Button>
          ))}
        </Card>
      </Section>
    </Page>
  );
}

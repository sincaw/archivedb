import React from "react";
import {CommandBar, FontIcon, ICommandBarItemProps, IStackStyles, mergeStyles, Stack} from "@fluentui/react";


export default function Header() {
  return <Stack styles={headerStackStyles}>
    <Stack horizontalAlign='center'>
      <CommandBar
        items={_items}
        farItems={_farItems}
        ariaLabel="Header actions"
      />
    </Stack>
  </Stack>
}

const _items: ICommandBarItemProps[] = [
  {
    key: 'weibo',
    text: 'Weibo',
    iconProps: {iconName: 'weibo'},
    onRenderIcon: (icon) => {
      return <FontIcon aria-label="OneDrive logo" iconName="weibo" className={highLightIcon}/>
    },
    onClick: () => console.log('Share'),
  },
  {
    key: 'zhihu',
    text: 'Zhihu',
    iconProps: {iconName: 'zhihu'},
    onClick: () => console.log('Zhihu'),
  }, {
    key: 'twitter',
    text: 'Twitter',
    iconProps: {iconName: 'twitter'},
    onClick: () => console.log('twitter'),
  },
];

const _farItems: ICommandBarItemProps[] = [
  {
    key: 'setting',
    text: 'Setting',
    // This needs an ariaLabel since it's icon-only
    ariaLabel: 'Setting',
    iconOnly: true,
    iconProps: {iconName: 'Settings'},
    onClick: () => console.log('Info'),
  },
];

const headerStackStyles: Partial<IStackStyles> = {
  root: {
    overflow: 'hidden',
    position: 'fixed',
    borderBottom: '1px solid',
    background: 'white',
    top: 0,
    width: '100%',
    zIndex: 777,
  }
}

const highLightIcon = mergeStyles({selectors: {'.content': {fill: '#1296db'}}})

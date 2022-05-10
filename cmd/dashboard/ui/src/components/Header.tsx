import React from "react";
import {CommandBar, FontIcon, ICommandBarItemProps, IStackStyles, mergeStyles, Stack} from "@fluentui/react";
import {useNavigate} from "react-router-dom";


export default function Header() {
  const navigate = useNavigate();
  const _farItems: ICommandBarItemProps[] = [
    {
      key: 'setting',
      text: 'Setting',
      // This needs an ariaLabel since it's icon-only
      ariaLabel: 'Setting',
      iconOnly: true,
      iconProps: {iconName: 'Settings'},
      onClick: () => navigate('/settings'),
    },
  ];

  const _items: ICommandBarItemProps[] = [
    {
      key: 'weibo',
      text: 'Weibo',
      iconProps: {iconName: 'weibo'},
      onRenderIcon: (icon) => {
        return <FontIcon aria-label="Weibo" iconName="weibo" className={highLightIcon}/>
      },
      onClick: () => navigate('/')
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

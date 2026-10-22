# Atualizações e Resolução de Problemas

Utilize esta página quando as instalações, atualizações ou arranques do Bedrock Nexus não se comportarem como esperado.

## Problemas comuns

### O Gaming Services está em falta ou danificado

- Repare-o ou instale-o através do fluxo guiado.
- Se necessário, reveja o estado de instalação do Minecraft Bedrock na Microsoft Store.

### O GameInput está em falta

- Instale o pacote redistribuível necessário quando o Bedrock Nexus o solicitar.
- Reinicie o launcher após a instalação, se necessário.

### O caminho de instalação não permite escrita

- Mova o caminho dos conteúdos geridos para uma localização com permissão de escrita nas **Definições**.
- Se o launcher pedir elevação para a instalação ou para a atualização automática, permita-o quando confiar na ação.

### O Bedrock Nexus fecha imediatamente ou nunca mostra uma janela

- No Windows, o Bedrock Nexus inicia agora o WebView2 num modo de renderização por software mais seguro, para reduzir falhas de arranque em algumas combinações de GPU e controladores.
- Se o launcher continuar a falhar antes de a interface aparecer, verifique `%APPDATA%\BedrockNexus.exe\logs\startup.log`.
- Tente de novo uma vez após um arranque falhado. Se continuar a acontecer, anexe o `startup.log` ao abrir um problema no GitHub.

### Uma versão falha ao arrancar

- teste a mesma versão sem mods
- confirme que os componentes do Windows necessários continuam a existir
- verifique se a versão terminou a instalação corretamente
- tente de novo com uma configuração de teste isolada e limpa

### As transferências são demasiado lentas ou falham repetidamente

- mudar de espelho
- tentar de novo mais tarde
- utilizar uma fonte de pacote local, se estiver disponível

## Passos de recuperação seguros

1. Faça uma cópia de segurança dos mundos importantes.
2. Remova mods ou pacotes recentes.
3. Teste com uma versão isolada limpa.
4. Reinstale a versão afetada, se necessário.

## Comportamento da atualização automática

O Bedrock Nexus pode verificar, transferir e instalar atualizações da aplicação. Alguns ambientes podem exigir permissão elevada se a pasta de instalação não permitir escrita.

## Quando comunicar um erro

Abra um problema no GitHub quando:

- conseguir reproduzir o problema de forma consistente
- o problema persistir após os passos básicos de recuperação
- o problema parecer específico do Bedrock Nexus, e não um problema geral de configuração do Windows

Ao comunicar, inclua:

- versão do Windows
- versão do Bedrock Nexus
- os passos exatos que seguiu
- capturas de ecrã ou registos, se estiverem disponíveis

Abra problemas aqui: [GitHub Issues](https://github.com/BedrockNexusLauncher/BedrockNexusLauncher/issues)

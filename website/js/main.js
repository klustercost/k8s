/* ═══════════════════════════════════════════════════════════
   SCROLL-TRIGGERED ANIMATIONS
   ═══════════════════════════════════════════════════════════ */

(function initScrollAnimations() {
  const elements = document.querySelectorAll('.animate-on-scroll');

  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          entry.target.classList.add('visible');
          observer.unobserve(entry.target);
        }
      });
    },
    { threshold: 0.1, rootMargin: '0px 0px -40px 0px' }
  );

  elements.forEach((el) => observer.observe(el));
})();

/* ═══════════════════════════════════════════════════════════
   NAVBAR SCROLL EFFECT
   ═══════════════════════════════════════════════════════════ */

(function initNavbar() {
  const navbar = document.getElementById('navbar');
  let ticking = false;

  window.addEventListener('scroll', () => {
    if (!ticking) {
      window.requestAnimationFrame(() => {
        navbar.classList.toggle('scrolled', window.scrollY > 50);
        ticking = false;
      });
      ticking = true;
    }
  });
})();

/* ═══════════════════════════════════════════════════════════
   MOBILE MENU TOGGLE
   ═══════════════════════════════════════════════════════════ */

(function initMobileMenu() {
  const btn = document.getElementById('mobile-menu-btn');
  const menu = document.getElementById('mobile-menu');

  btn.addEventListener('click', () => {
    menu.classList.toggle('open');
  });

  menu.querySelectorAll('a').forEach((link) => {
    link.addEventListener('click', () => {
      menu.classList.remove('open');
    });
  });
})();

/* ═══════════════════════════════════════════════════════════
   SMOOTH SCROLL FOR ANCHOR LINKS
   ═══════════════════════════════════════════════════════════ */

(function initSmoothScroll() {
  const NAVBAR_HEIGHT = 96;

  document.querySelectorAll('a[href^="#"]').forEach((anchor) => {
    anchor.addEventListener('click', (e) => {
      const targetId = anchor.getAttribute('href');
      if (targetId === '#') return;

      const target = document.querySelector(targetId);
      if (!target) return;

      e.preventDefault();

      const menu = document.getElementById('mobile-menu');
      if (menu.classList.contains('open')) {
        menu.classList.remove('open');
      }

      const top = target.getBoundingClientRect().top + window.scrollY - NAVBAR_HEIGHT;
      window.scrollTo({ top, behavior: 'smooth' });
    });
  });
})();

/* ═══════════════════════════════════════════════════════════
   TYPING ANIMATION (AI QUERIES SECTION)
   ═══════════════════════════════════════════════════════════ */

(function initTypingEffect() {
  const queries = [
    {
      question: 'Which pod consumed the most CPU in the last hour?',
      natural: 'The api-gateway pod in production used the most CPU at 845 millicores, followed by worker-batch in jobs at 612m.',
      response:
        '[\n  { "pod": "api-gateway", "cpu": "845m", "namespace": "prod" },\n  { "pod": "worker-batch",  "cpu": "612m", "namespace": "jobs" }\n]',
    },
    {
      question: 'What is the total cost per namespace this week?',
      natural: 'Production is your most expensive namespace at $142.38 this week. Staging costs $47.20 and jobs is at $31.05.',
      response:
        '[\n  { "namespace": "production", "cost": "$142.38" },\n  { "namespace": "staging",    "cost": "$47.20" },\n  { "namespace": "jobs",       "cost": "$31.05" }\n]',
    },
    {
      question: 'Show nodes with no price assigned yet',
      natural: 'One node is missing pricing data: aks-pool2-vm3 running a Standard_D4s_v3 SKU. It may need a manual price entry.',
      response:
        '[\n  { "node": "aks-pool2-vm3", "sku": "Standard_D4s_v3", "price": 0}\n]',
    },
  ];

  const typedEl = document.getElementById('typed-query');
  const cursorEl = document.getElementById('cursor-blink');
  const responseEl = document.getElementById('typed-response');
  const responseContent = document.getElementById('response-content');
  const responseNatural = document.getElementById('response-natural');

  if (!typedEl) return;

  const dynamicArea = document.getElementById('ai-dynamic-area');

  function lockHeight() {
    dynamicArea.style.minHeight = '';
    responseEl.style.visibility = 'visible';
    responseEl.style.opacity = '1';
    responseEl.style.position = '';

    let maxH = 0;
    queries.forEach((q) => {
      typedEl.textContent = q.question;
      responseNatural.textContent = q.natural;
      responseContent.textContent = q.response;
      maxH = Math.max(maxH, dynamicArea.offsetHeight);
    });

    dynamicArea.style.minHeight = maxH + 'px';

    typedEl.textContent = '';
    responseNatural.textContent = '';
    responseEl.style.visibility = '';
    responseEl.style.opacity = '';
    responseEl.classList.add('invisible', 'opacity-0');
  }

  lockHeight();
  window.addEventListener('resize', lockHeight);

  let queryIndex = 0;
  let charIndex = 0;
  let isDeleting = false;
  let isPausing = false;

  function type() {
    const current = queries[queryIndex];

    if (isPausing) return;

    if (!isDeleting) {
      typedEl.textContent = current.question.substring(0, charIndex + 1);
      charIndex++;

      if (charIndex === current.question.length) {
        isPausing = true;
        cursorEl.style.display = 'none';

        setTimeout(() => {
          responseNatural.textContent = current.natural;
          responseContent.textContent = current.response;
          responseEl.classList.remove('invisible', 'opacity-0');
        }, 300);

        setTimeout(() => {
          isDeleting = true;
          isPausing = false;
          cursorEl.style.display = '';
          type();
        }, 3500);
        return;
      }

      setTimeout(type, 35 + Math.random() * 25);
    } else {
      responseEl.classList.add('invisible', 'opacity-0');
      typedEl.textContent = current.question.substring(0, charIndex - 1);
      charIndex--;

      if (charIndex === 0) {
        isDeleting = false;
        queryIndex = (queryIndex + 1) % queries.length;
        setTimeout(type, 500);
        return;
      }

      setTimeout(type, 15);
    }
  }

  const aiSection = document.getElementById('ai-queries');
  const aiObserver = new IntersectionObserver(
    (entries) => {
      if (entries[0].isIntersecting) {
        setTimeout(type, 600);
        aiObserver.disconnect();
      }
    },
    { threshold: 0.3 }
  );
  aiObserver.observe(aiSection);
})();

/* ═══════════════════════════════════════════════════════════
   COPY CODE BUTTON
   ═══════════════════════════════════════════════════════════ */

function copyCode(button) {
  const codeBlock = button.closest('.glass, .rounded-xl').querySelector('code');
  if (!codeBlock) return;

  navigator.clipboard.writeText(codeBlock.textContent.trim()).then(() => {
    const original = button.innerHTML;
    button.innerHTML =
      '<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg> Copied!';
    button.classList.add('text-emerald-400');

    setTimeout(() => {
      button.innerHTML = original;
      button.classList.remove('text-emerald-400');
    }, 2000);
  });
}

/* ═══════════════════════════════════════════════════════════
   ACTIVE NAV LINK HIGHLIGHT
   ═══════════════════════════════════════════════════════════ */

(function initActiveNav() {
  const sections = document.querySelectorAll('section[id]');
  const navLinks = document.querySelectorAll('nav .nav-link');

  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          const id = entry.target.getAttribute('id');
          navLinks.forEach((link) => {
            const isActive = link.getAttribute('href') === '#' + id;
            link.classList.toggle('text-white', isActive);
            link.classList.toggle('text-slate-400', !isActive);
          });
        }
      });
    },
    { threshold: 0.3, rootMargin: '-80px 0px -50% 0px' }
  );

  sections.forEach((section) => observer.observe(section));
})();

/* ═══════════════════════════════════════════════════════════
   HANNOVER MESSE POPUP
   ═══════════════════════════════════════════════════════════ */

(function initMessePopup() {
  const STORAGE_KEY = 'messe-popup-dismissed';
  const popup = document.getElementById('messe-popup');
  if (!popup) return;

  if (localStorage.getItem(STORAGE_KEY)) return;

  setTimeout(() => {
    popup.classList.add('active');
    document.body.style.overflow = 'hidden';
  }, 1500);

  function close() {
    popup.classList.remove('active');
    document.body.style.overflow = '';
    localStorage.setItem(STORAGE_KEY, Date.now());
  }

  document.getElementById('messe-close').addEventListener('click', close);
  document.getElementById('messe-backdrop').addEventListener('click', close);

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && popup.classList.contains('active')) close();
  });
})();
